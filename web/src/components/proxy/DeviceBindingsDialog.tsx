import { DesktopRegular, LinkRegular } from "@fluentui/react-icons";
import type { DeviceListItem, DeviceProxyBinding, UpstreamProxy } from "../../types";
import { Button, EmptyState, Modal, Tag } from "../ui";
import { useI18n } from "../../lib/i18n";

export interface DeviceBindingsDialogProps {
  open: boolean;
  proxy: UpstreamProxy | null;
  proxies: UpstreamProxy[];
  devices: DeviceListItem[];
  bindings: DeviceProxyBinding[];
  busyDevice: string;
  onBind: (deviceId: string) => void;
  onUnbind: (deviceId: string) => void;
  onClose: () => void;
}

export function DeviceBindingsDialog(props: DeviceBindingsDialogProps) {
  const { t } = useI18n();
  const { open, proxy, proxies, devices, bindings, busyDevice, onBind, onUnbind, onClose } = props;
  const proxyName = proxy?.name || proxy?.id || "";
  const bindingByDevice = new Map(bindings.map((item) => [item.deviceId, item]));
  const proxyNameById = new Map(proxies.map((item) => [item.id, item.name || item.id]));

  return (
    <Modal open={open} onClose={onClose} title={`${t("设备绑定")} — ${proxyName}`} width="max-w-2xl">
      <div className="space-y-4 pb-2">
        <div className="rounded-lg border border-sky-200/70 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-800/50 dark:bg-sky-900/20 dark:text-sky-200">
          {t("绑定后，该设备的 VoWiFi 建链和通信都会使用此 SOCKS5 代理；解绑后恢复直连。配置变更会立即尝试重连 VoWiFi。")}
        </div>
        {devices.length === 0 ? (
          <EmptyState title={t("暂无可绑定设备")} subtitle={t("请先在设备管理中添加设备。")}/>
        ) : (
          <div className="space-y-2">
            {devices.map((device) => {
              const binding = bindingByDevice.get(device.id);
              const boundHere = binding?.upstreamProxyId === proxy?.id;
              const boundElsewhere = !!binding && !boundHere;
              return (
                <div key={device.id} className="ui-panel-muted flex items-center justify-between gap-3 rounded-lg p-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white text-sky-600 shadow-sm dark:bg-white/10 dark:text-sky-300">
                      <DesktopRegular className="text-[18px]" />
                    </span>
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="truncate text-sm font-semibold text-gray-900 dark:text-white">{device.name || device.id}</span>
                        <span className="font-mono text-xs text-gray-400">{device.id}</span>
                        {boundHere ? <Tag type="success">{t("已绑定")}</Tag> : null}
                        {!device.vowifiEnabled ? <Tag type="info">{t("VoWiFi 未启用")}</Tag> : null}
                      </div>
                      <div className="mt-0.5 text-xs text-gray-500">
                        {boundHere
                          ? t("当前通过此代理通信")
                          : boundElsewhere
                            ? `${t("当前绑定")}: ${proxyNameById.get(binding.upstreamProxyId) || binding.upstreamProxyId}`
                            : t("当前直连")}
                      </div>
                    </div>
                  </div>
                  {boundHere ? (
                    <Button size="small" variant="danger" loading={busyDevice === device.id} onClick={() => onUnbind(device.id)}>
                      {t("解绑")}
                    </Button>
                  ) : (
                    <Button size="small" variant="primary" icon={<LinkRegular />} loading={busyDevice === device.id} onClick={() => onBind(device.id)}>
                      {boundElsewhere ? t("切换绑定") : t("绑定设备")}
                    </Button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </Modal>
  );
}
