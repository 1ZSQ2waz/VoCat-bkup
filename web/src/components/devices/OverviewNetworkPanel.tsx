import { FieldRow } from "./FieldRow";
import type { DeviceDetail } from "./types";
import { useI18n } from "../../lib/i18n";

export interface OverviewNetworkPanelProps {
  device: DeviceDetail;
  trafficMinuteRx: string;
  trafficMinuteTx: string;
  trafficSpeedRx: string;
  trafficSpeedTx: string;
}

export function OverviewNetworkPanel({ device, trafficMinuteRx, trafficMinuteTx, trafficSpeedRx, trafficSpeedTx }: OverviewNetworkPanelProps) {
  const { t } = useI18n();
  const traffic = device.traffic || {};
  const metaStatus = device.trafficMeta?.status;
  const sampleNote = metaStatus === "waiting_sample" ? t("等待采样") : metaStatus === "stale" ? t("采样中断") : "";
  const off = t("数据未开启");

  const minuteRx = trafficMinuteRx || sampleNote || traffic.rx;
  const minuteTx = trafficMinuteTx || sampleNote || traffic.tx;
  const speedRx = trafficSpeedRx || sampleNote || traffic.rate || "--";
  const speedTx = trafficSpeedTx || sampleNote || "--";

  return (
    <div className="ui-panel-muted p-4">
      <div className="mb-2 text-xs font-bold uppercase tracking-wider text-gray-500">{t("网络")}</div>
      {off ? (
        <div className="flex items-center justify-center p-6 text-sm text-gray-400">{off}</div>
      ) : (
        <div className="space-y-1.5 text-sm text-gray-700 dark:text-gray-200">
          <FieldRow label={t("内网 IPv4")} value={device.privateIp} monospace copyable />
          <FieldRow label={t("内网 IPv6")} value={device.privateIpv6} monospace copyable />
          <FieldRow label={t("近1分钟上传")} value={minuteTx} monospace />
          <FieldRow label={t("近1分钟下载")} value={minuteRx} monospace />
          <FieldRow label={t("实时下载速率")} value={speedRx} monospace />
          <FieldRow label={t("实时上传速率")} value={speedTx} monospace />
        </div>
      )}
    </div>
  );
}
