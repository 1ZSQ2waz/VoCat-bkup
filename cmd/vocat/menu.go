package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"

	"vocat/internal/auth"
	"vocat/internal/config"
	"vocat/internal/store"
)

//envFilePath is the systemd EnvironmentFile that carries VOCAT_ADMIN_PASSWORD.
// EnsureAdmin reseeds the DB from it on every start, so change-password must
// rewrite it or the next restart reverts the password.
const envFilePath = "/etc/vocat/env"

const systemdUnitPath = "/etc/systemd/system/vocat.service"

// runMenu is the interactive lifecycle menu: change password, restart the
// systemd unit, or fully uninstall vocat. It must run as root on the host
// (needs systemctl + the 0600 env file). Docker deployments do not use it.
func runMenu(logger *slog.Logger) error {
	if os.Geteuid() != 0 {
		return errors.New("vocat menu must run as root (needs systemctl and /etc/vocat/env)")
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("vocat menu requires an interactive terminal")
	}

	lang := promptLanguage()
	menu := newMenu(lang)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		fmt.Println(menu.title())
		for _, opt := range menu.options() {
			fmt.Printf("  %s\n", opt)
		}
		fmt.Print(menu.prompt())
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read menu choice: %w", err)
		}
		choice := strings.TrimSpace(line)
		switch choice {
		case "1":
			if err := menuChangePassword(reader, menu, logger); err != nil {
				fmt.Println(menu.errorPrefix(err))
			}
		case "2":
			if err := menuRestart(menu); err != nil {
				fmt.Println(menu.errorPrefix(err))
			}
		case "3":
			if err := menuUninstall(reader, menu); err != nil {
				fmt.Println(menu.errorPrefix(err))
			}
		case "0", "":
			fmt.Println(menu.bye())
			return nil
		default:
			fmt.Println(menu.invalid())
		}
	}
}

// promptLanguage asks for 中文 (1) or English (2) once per invocation. The
// user chose to re-ask every run rather than persist a language preference.
func promptLanguage() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("选择语言 / Select language:  1) 中文   2) English")
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "zh"
		}
		switch strings.TrimSpace(line) {
		case "1", "":
			return "zh"
		case "2":
			return "en"
		}
	}
}

func menuChangePassword(reader *bufio.Reader, m *menu, logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("%w: %v", errMenuConfig, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("%w: %v", errMenuStore, err)
	}
	defer database.Close()

	authService, err := auth.New(database, auth.Options{SessionTTL: cfg.SessionTTL})
	if err != nil {
		return fmt.Errorf("%w: %v", errMenuAuth, err)
	}

	fmt.Print(m.currentPassword())
	currentPw, err := readPasswordMasked()
	if err != nil {
		return err
	}
	fmt.Print(m.newPassword())
	newPw, err := readPasswordMasked()
	if err != nil {
		return err
	}
	fmt.Print(m.confirmPassword())
	confirmPw, err := readPasswordMasked()
	if err != nil {
		return err
	}
	fmt.Println()
	if newPw != confirmPw {
		return errPasswordsDiffer
	}
	if err := authService.ChangePassword(ctx, cfg.AdminUsername, currentPw, newPw); err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return errCurrentWrong
		}
		return fmt.Errorf("%w: %v", errMenuAuth, err)
	}
	// Persist the new plaintext to the env file so the next EnsureAdmin (on
	// restart) agrees with the hash we just wrote to the DB. Without this the
	// restart reverts the password to whatever the env file still holds.
	if err := rewriteEnvPassword(newPw); err != nil {
		logger.Error("menu: password changed in DB but env file rewrite failed; restart will revert", "error", err)
		return fmt.Errorf("%w: %v", errMenuEnvWrite, err)
	}
	fmt.Println(m.passwordChanged())
	return nil
}

// readPasswordMasked reads a password with echo disabled. term.ReadPassword
// does not return the trailing newline, so we print one for a clean prompt.
func readPasswordMasked() (string, error) {
	fd := int(os.Stdin.Fd())
	bytes, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(bytes), nil
}

// rewriteEnvPassword replaces (or appends) the VOCAT_ADMIN_PASSWORD line in the
// systemd EnvironmentFile and keeps the file 0600. The replacement is atomic:
// the temp file lives in the same directory so os.Rename stays on one
// filesystem.
func rewriteEnvPassword(newPassword string) error {
	const key = "VOCAT_ADMIN_PASSWORD="
	var lines []string
	if data, err := os.ReadFile(envFilePath); err == nil {
		lines = strings.Split(string(data), "\n")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, key) {
			lines[i] = key + newPassword
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, key+newPassword)
	}
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	dir := envFilePath[:strings.LastIndex(envFilePath, "/")]
	tmp, err := os.CreateTemp(dir, ".vocat-env-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, envFilePath)
}

func menuRestart(m *menu) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return errNoSystemctl
	}
	cmd := exec.Command("systemctl", "restart", "vocat")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", errRestartFailed, strings.TrimSpace(string(out)))
	}
	fmt.Println(m.restarted())
	return nil
}

// menuUninstall performs full removal: stop/disable the unit, delete the unit,
// remove /opt/vocat (binary + data + SQLite DB), remove the env file, reload
// systemd, and best-effort delete the vocat user.
func menuUninstall(reader *bufio.Reader, m *menu) error {
	fmt.Println(m.uninstallWarn())
	fmt.Print(m.uninstallConfirm())
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if strings.TrimSpace(line) != "yes" {
		fmt.Println(m.uninstallCancelled())
		return nil
	}

	runIgnore := func(name string, args ...string) {
		_ = exec.Command(name, args...).Run()
	}
	runIgnore("systemctl", "stop", "vocat")
	runIgnore("systemctl", "disable", "vocat")
	_ = os.Remove(systemdUnitPath)
	_ = os.RemoveAll("/opt/vocat")
	_ = os.Remove(envFilePath)
	_ = os.Remove("/etc/vocat") // succeeds only when empty
	runIgnore("systemctl", "daemon-reload")
	runIgnore("userdel", "vocat")

	fmt.Println(m.uninstalled())
	return nil
}

// menu-local sentinel errors so callers can map them to localized messages.
var (
	errCurrentWrong    = errors.New("menu: current password is incorrect")
	errPasswordsDiffer = errors.New("menu: passwords do not match")
	errNoSystemctl     = errors.New("menu: systemctl not found")
	errRestartFailed   = errors.New("menu: restart failed")
	errMenuConfig      = errors.New("menu: load configuration")
	errMenuStore       = errors.New("menu: open database")
	errMenuAuth        = errors.New("menu: auth service")
	errMenuEnvWrite    = errors.New("menu: write env file")
)

// ---- i18n ----

type menu struct{ lang string }

func newMenu(lang string) *menu { return &menu{lang: lang} }

// msg returns the localized string for a key. Each key carries [zh, en].
func (m *menu) msg(key string) string {
	const zh, en = 0, 1
	table := map[string][2]string{
		"title":      {"vocat 管理菜单", "vocat management menu"},
		"opt_change": {"1) 修改密码", "1) Change password"},
		"opt_restart": {"2) 重启服务", "2) Restart service"},
		"opt_uninstall": {"3) 卸载程序", "3) Uninstall"},
		"opt_exit":   {"0) 退出", "0) Exit"},
		"prompt":     {"请选择: ", "Select: "},
		"invalid":    {"无效选项，请重试。", "Invalid choice, try again."},
		"bye":        {"再见。", "Bye."},
		"cur_pw":     {"当前密码: ", "Current password: "},
		"new_pw":     {"新密码 (至少 12 位): ", "New password (min 12 chars): "},
		"confirm_pw": {"确认新密码: ", "Confirm new password: "},
		"pw_changed": {"密码已修改。重启后仍然有效。", "Password changed. Survives restart."},
		"restarted":  {"服务已重启。", "Service restarted."},
		"uninstall_warn": {
			"警告: 将删除程序、数据与配置,且不可恢复!",
			"WARNING: removes the program, data and config. Irreversible!",
		},
		"uninstall_confirm": {"输入 yes 确认卸载: ", "Type yes to confirm uninstall: "},
		"uninstall_cancelled": {"已取消卸载。", "Uninstall cancelled."},
		"uninstalled": {"vocat 已卸载。", "vocat uninstalled."},
	}
	entry, ok := table[key]
	if !ok {
		return key
	}
	if m.lang == "en" {
		return entry[en]
	}
	return entry[zh]
}

func (m *menu) title() string      { return m.msg("title") }
func (m *menu) prompt() string     { return m.msg("prompt") }
func (m *menu) invalid() string    { return m.msg("invalid") }
func (m *menu) bye() string        { return m.msg("bye") }
func (m *menu) currentPassword() string  { return m.msg("cur_pw") }
func (m *menu) newPassword() string      { return m.msg("new_pw") }
func (m *menu) confirmPassword() string  { return m.msg("confirm_pw") }
func (m *menu) passwordChanged() string  { return m.msg("pw_changed") }
func (m *menu) restarted() string        { return m.msg("restarted") }
func (m *menu) uninstallWarn() string    { return m.msg("uninstall_warn") }
func (m *menu) uninstallConfirm() string { return m.msg("uninstall_confirm") }
func (m *menu) uninstallCancelled() string { return m.msg("uninstall_cancelled") }
func (m *menu) uninstalled() string      { return m.msg("uninstalled") }

func (m *menu) options() []string {
	return []string{m.msg("opt_change"), m.msg("opt_restart"), m.msg("opt_uninstall"), m.msg("opt_exit")}
}

func (m *menu) errorPrefix(err error) string {
	switch {
	case errors.Is(err, errCurrentWrong):
		if m.lang == "en" {
			return "Current password is incorrect."
		}
		return "当前密码不正确。"
	case errors.Is(err, errPasswordsDiffer):
		if m.lang == "en" {
			return "Passwords do not match."
		}
		return "两次输入的密码不一致。"
	case errors.Is(err, errNoSystemctl):
		if m.lang == "en" {
			return "systemctl not found."
		}
		return "未找到 systemctl。"
	case errors.Is(err, errRestartFailed):
		if m.lang == "en" {
			return "Restart failed."
		}
		return "重启失败。"
	case errors.Is(err, errMenuConfig):
		if m.lang == "en" {
			return "Failed to load configuration."
		}
		return "加载配置失败。"
	case errors.Is(err, errMenuStore):
		if m.lang == "en" {
			return "Failed to open the database."
		}
		return "打开数据库失败。"
	case errors.Is(err, errMenuAuth):
		if m.lang == "en" {
			return "Auth service error."
		}
		return "认证服务错误。"
	case errors.Is(err, errMenuEnvWrite):
		if m.lang == "en" {
			return "Password changed in DB, but the env file rewrite failed — restart will revert it. Check " + envFilePath + "."
		}
		return "数据库密码已修改，但环境变量文件写入失败——重启后将回滚。请检查 " + envFilePath + "。"
	default:
		if m.lang == "en" {
			return "Error: " + err.Error()
		}
		return "错误: " + err.Error()
	}
}
