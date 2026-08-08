#!/usr/bin/env python3
"""Run a command on the vocat test machine over SSH (paramiko, password auth)."""
import base64
import sys

import paramiko

HOST, USER, PASSWORD = "192.168.2.222", "root", "lammy520"


def main() -> int:
    if len(sys.argv) > 2 and sys.argv[1] == "--base64":
        command = base64.b64decode(sys.argv[2]).decode("utf-8")
    else:
        command = sys.argv[1] if len(sys.argv) > 1 else "uname -a"
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASSWORD, timeout=10)
    _, stdout, stderr = client.exec_command(command, timeout=60)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    sys.stdout.write(out)
    if err:
        sys.stderr.write(err)
    return code


if __name__ == "__main__":
    sys.exit(main())
