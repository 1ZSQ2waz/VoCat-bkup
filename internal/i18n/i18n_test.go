package i18n

import "testing"

func TestDefaultIsChinese(t *testing.T) {
	Set("zh") // ensure a clean Chinese baseline regardless of test order
	if got := T("已启用"); got != "已启用" {
		t.Fatalf("zh: T(已启用) = %q, want 已启用", got)
	}
}

func TestEnglishTranslation(t *testing.T) {
	Set("en")
	defer Set("zh")
	if got := T("已启用"); got != "Enabled" {
		t.Fatalf("en: T(已启用) = %q, want Enabled", got)
	}
	if got := T("美国"); got != "United States" {
		t.Fatalf("en: T(美国) = %q, want United States", got)
	}
	// unknown strings fall back to the Chinese key unchanged
	if got := T("未收录的字符串"); got != "未收录的字符串" {
		t.Fatalf("en: unknown string = %q, want fallback unchanged", got)
	}
}

func TestTfInterpolation(t *testing.T) {
	Set("en")
	defer Set("zh")
	got := Tf("设备数量已达上限，最多只能添加 %d 台设备", 5)
	want := "Device limit reached; at most 5 devices can be added."
	if got != want {
		t.Fatalf("Tf = %q, want %q", got, want)
	}
	// region reason: country name itself is translated before interpolation
	got = Tf("SIM 卡归属地为%s（MCC %s），本服务不向该地区卡片提供数据/短信/VoWiFi", T("中国"), "460")
	want = "The SIM's home region is China (MCC 460); this service does not provide data, SMS, or VoWiFi to cards from that region."
	if got != want {
		t.Fatalf("Tf region = %q, want %q", got, want)
	}
}

func TestSetNormalizesUnknownToEnglish(t *testing.T) {
	Set("fr") // unsupported -> treated as English
	defer Set("zh")
	if Lang() != "en" {
		t.Fatalf("Lang after Set(fr) = %q, want en", Lang())
	}
}
