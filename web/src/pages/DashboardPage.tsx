import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import type { DashboardDevice } from "../types";
import { usePolling } from "../lib/usePolling";
import { useI18n } from "../lib/i18n";
import { PageHeader } from "../components/ui/PageHeader";
import { RefreshButton } from "../components/ui/RefreshButton";
import { ErrorState } from "../components/ui/ErrorState";
import { ListSkeleton } from "../components/ui/ListSkeleton";
import { EmptyState } from "../components/ui/EmptyState";
import { DeviceCard } from "../components/DeviceCard";

interface LoadError { message: string; status?: number; method?: string; url?: string }

export default function DashboardPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [devices, setDevices] = useState<DashboardDevice[]>([]);
  const [devicesLoading, setDevicesLoading] = useState(false);
  const [devicesError, setDevicesError] = useState<LoadError | null>(null);
  const [devicesOkAt, setDevicesOkAt] = useState<number | null>(null);
  const [lastRefresh, setLastRefresh] = useState<number | null>(null);

  const fetchDevices = useCallback(async () => {
    setDevicesLoading(true);
    try {
      const list = await api<DashboardDevice[]>("/dashboard/devices");
      setDevices(list || []);
      setDevicesError(null);
      const now = Date.now();
      setDevicesOkAt(now);
      setLastRefresh(now);
    } catch (e: any) {
      setDevicesError({ message: e?.message || t("加载失败"), status: e?.status });
    } finally {
      setDevicesLoading(false);
    }
  }, []);

  usePolling(fetchDevices, 5000);

  const total = devices.length;
  const online = devices.filter((d) => d?.healthy).length;
  const offline = Math.max(0, total - online);
  const openDevice = (id: string) => navigate(`/devices?device=${encodeURIComponent(id)}&tab=overview`);

  return (
    <div>
      <PageHeader
        title={t("设备监控")}
        subtitle={t("实时监测模组检测状态与出口连通性")}
        actions={<RefreshButton loading={devicesLoading} onClick={fetchDevices} />}
      />
      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="ui-panel p-4"><div className="text-xs text-gray-400">{t("设备总数")}</div><div className="mt-1 text-2xl font-extrabold">{total}</div></div>
        <div className="ui-panel p-4"><div className="text-xs text-gray-400">{t("在线")}</div><div className="mt-1 text-2xl font-extrabold text-green-600 dark:text-green-400">{online}</div></div>
        <div className="ui-panel p-4"><div className="text-xs text-gray-400">{t("离线")}</div><div className="mt-1 text-2xl font-extrabold text-red-600 dark:text-red-400">{offline}</div></div>
        <div className="ui-panel p-4"><div className="text-xs text-gray-400">{t("最近刷新")}</div><div className="mt-2 font-mono text-sm text-gray-600 dark:text-gray-300">{lastRefresh ? new Date(lastRefresh).toLocaleTimeString() : "--:--:--"}</div></div>
      </div>
      {devicesError ? (
        <ErrorState className="mb-6" title={t("设备列表加载失败")} message={devicesError.message} statusCode={devicesError.status} requestMethod={devicesError.method} requestUrl={devicesError.url} lastSuccessAt={devicesOkAt} retryText={t("重试")} onRetry={fetchDevices} />
      ) : null}
      {devicesLoading && devices.length === 0 ? (
        <ListSkeleton rows={10} />
      ) : devices.length === 0 ? (
        <EmptyState title={t("暂无设备接入")} subtitle={t("请先在设备管理中添加或接管设备")} />
      ) : (
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
          {devices.map((d) => (
            <DeviceCard key={d.id} device={d} onOpen={openDevice} />
          ))}
        </div>
      )}
    </div>
  );
}
