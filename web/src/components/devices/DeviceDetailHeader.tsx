import { ArrowSyncRegular, PowerRegular, ChatRegular } from "@fluentui/react-icons";
import { Button } from "../ui";
import type { DeviceDetail } from "./types";
import { useI18n } from "../../lib/i18n";
import { isEC20Model } from "../../lib/utils";

export interface DeviceDetailHeaderProps {
  device: DeviceDetail;
  rotating: boolean;
  rebooting: boolean;
  reconnectingVoWiFi: boolean;
  onCopyText: (text: string) => void;
  onRotateIp: () => void;
  onReconnectVowifi: () => void;
  onRebootModem: () => void;
  onOpenSms: () => void;
}

export function DeviceDetailHeader(props: DeviceDetailHeaderProps) {
  const { t } = useI18n();
  const { device } = props;
	const vowifiInUse = device.vowifiEnabled || device.vowifiActive || device.vowifiRuntime?.smsReady;
  const brandImg = isEC20Model(device.modem?.model);
  return (
    <div className="ui-card p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-3">
            {brandImg ? (
              <img src="/ec20.png" alt="" className="h-11 w-11 flex-shrink-0 object-contain" />
            ) : (
              <div className="device-header-brand-icon">
                <svg viewBox="0 0 1025 1024" width="200" height="200" className="device-header-brand-svg" aria-hidden="true">
                  <path d="M512.473172 1023.995242A511.814852 511.814852 0 0 1 313.545134 40.351073a512.244696 512.244696 0 0 1 398.855715 943.658633 508.815937 508.815937 0 0 1-199.927677 39.985536z m0-943.658634C274.559237 80.336608 80.629391 274.266455 80.629391 512.18039s193.929846 431.843781 431.843781 431.843781 431.843781-193.929846 431.843781-431.843781S751.386745 80.336608 512.473172 80.336608z" />
                  <path d="M506.475342 716.10662a39.985535 39.985535 0 0 1-39.985536-39.985535v-76.972156c0-79.971071 64.976495-144.947566 144.947566-144.947565a77.971794 77.971794 0 0 0 0-155.943588H445.4974a56.979388 56.979388 0 0 0-56.979387 56.979388 39.985535 39.985535 0 0 1-79.971071 0c0-74.972879 60.977941-136.950458 136.950458-136.950459h164.940333c86.968539 0 157.942864 70.974325 157.942865 157.942865s-69.974687 157.942864-157.942865 157.942864a64.976495 64.976495 0 0 0-64.976494 64.976495v76.972156a39.985535 39.985535 0 0 1-38.985897 39.985535zM505.475703 742.097218a48.982281 48.982281 0 1 0 48.982281 48.982281 48.982281 48.982281 0 0 0-48.982281-48.982281z" />
                </svg>
              </div>
            )}
            <div className="min-w-0">
              <div className="truncate text-xl font-extrabold text-gray-900 dark:text-white">{device.name || device.id}</div>
              <div className="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
                <span className="cursor-pointer font-mono hover:underline" onClick={() => props.onCopyText(device.id)}>
                  {device.id}
                </span>
              </div>
            </div>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
		  {vowifiInUse ? (
            <Button loading={props.reconnectingVoWiFi} onClick={props.onReconnectVowifi} className="ui-glass-border !border-0" icon={<ArrowSyncRegular />}>
              {t("重连 VoWiFi")}
            </Button>
          ) : (
            <Button
              loading={props.rotating}
              disabled={!device?.networkConnected}
              onClick={props.onRotateIp}
              className="ui-glass-border !border-0"
              icon={<ArrowSyncRegular />}
            >
              {t("切换 IP")}
            </Button>
          )}
          <Button loading={props.rebooting} onClick={props.onRebootModem} className="ui-glass-border !border-0 hover:!text-red-600" icon={<PowerRegular />}>
            {t("重启模组")}
          </Button>
          <Button onClick={props.onOpenSms} className="ui-glass-border !border-0" icon={<ChatRegular />}>
            {t("短信")}
          </Button>
        </div>
      </div>
    </div>
  );
}
