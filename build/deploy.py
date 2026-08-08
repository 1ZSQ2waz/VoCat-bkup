#!/usr/bin/env python3
"""Deploy the freshly built vocat binary to the test machine and restart the service."""
import time

import paramiko

HOST, USER, PASSWORD = "192.168.2.222", "root", "lammy520"
LOCAL_BINARY = r"E:\tool\vocat\build\vocat-linux-amd64"
REMOTE_TMP = "/opt/vocat/bin/vocat.new"
REMOTE_BIN = "/opt/vocat/bin/vocat"


def run(client, command, timeout=60):
    _, stdout, stderr = client.exec_command(command, timeout=timeout)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    code = stdout.channel.recv_exit_status()
    if out.strip():
        print(out)
    if err.strip():
        print(err)
    if code != 0:
        raise SystemExit(f"command failed ({code}): {command}")


def main():
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASSWORD, timeout=10)

    stamp = time.strftime("%Y%m%d-%H%M%S")
    # Use SQLite's online backup command so WAL contents are included.
    database_backup = f"/opt/vocat/data/vocat.db.bak-deploy-{stamp}"
    run(client, f'''sqlite3 /opt/vocat/data/vocat.db ".backup '{database_backup}'"''')
    sftp = client.open_sftp()
    print(f"uploading {LOCAL_BINARY} -> {REMOTE_TMP} ...")
    sftp.put(LOCAL_BINARY, REMOTE_TMP)
    sftp.close()

    run(client, f"chmod 0755 {REMOTE_TMP}")
    run(client, f"cp -a {REMOTE_BIN} {REMOTE_BIN}.bak-deploy-{stamp}")
    run(client, f"mv {REMOTE_TMP} {REMOTE_BIN}")
    run(client, "systemctl restart vocat")
    time.sleep(2)
    run(client, "systemctl is-active vocat && systemctl show vocat -p MainPID --value")
    run(client, "curl -fsS http://127.0.0.1:7575/api/health || true")
    client.close()
    print("deploy OK")


if __name__ == "__main__":
    main()
