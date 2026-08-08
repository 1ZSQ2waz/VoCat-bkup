import { AddRegular, DeleteRegular, DesktopRegular, EditRegular, GlobeRegular } from "@fluentui/react-icons";
import type { UpstreamProxy } from "../../types";
import { Button, EmptyState, ErrorState, ListSkeleton, Tag } from "../ui";
import type { LoadError, UpstreamRow } from "./shared";
import { useI18n } from "../../lib/i18n";

export interface UpstreamSectionProps {
  rows: UpstreamRow[];
  loading: boolean;
  error: LoadError | null;
  onRetry: () => void;
  onNew: () => void;
  onEdit: (proxy: UpstreamProxy) => void;
  onDelete: (proxy: UpstreamProxy) => void;
  onOpenBindings: (proxy: UpstreamProxy) => void;
}

function UpstreamRowCard({
  row,
  onEdit,
  onDelete,
  onOpenBindings,
}: {
  row: UpstreamRow;
  onEdit: (proxy: UpstreamProxy) => void;
  onDelete: (proxy: UpstreamProxy) => void;
  onOpenBindings: (proxy: UpstreamProxy) => void;
}) {
  const { t } = useI18n();
  return (
    <div className="ui-panel-muted flex flex-col gap-3 p-4 lg:flex-row lg:items-center lg:justify-between">
      <div className="flex min-w-0 items-center gap-3">
        <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${row.enabled ? "bg-green-500" : "bg-gray-300"}`} />
        <div className="min-w-0">
          <div className="truncate font-bold text-gray-900 dark:text-white">{row.name || row.id}</div>
          <div className="mt-0.5 truncate text-xs text-gray-500">
            SOCKS5 · <span className="font-mono">{row.addr}</span>
            {row.username ? <span> · {t("鉴权")}: {row.username}</span> : null}
          </div>
        </div>
      </div>
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        <Tag type={row.enabled ? "success" : "info"}>{row.enabled ? t("已启用") : t("已禁用")}</Tag>
        <div className="inline-flex items-center gap-1 rounded border border-indigo-200/60 bg-indigo-50 px-2 py-0.5 text-[11px] font-medium text-indigo-600 dark:border-indigo-800/40 dark:bg-indigo-900/20 dark:text-indigo-400">
          <DesktopRegular className="text-[14px]" />
          <span>{row.bindingCount} {t("台设备")}</span>
        </div>
        <div className="mx-0.5 hidden h-3.5 w-px bg-gray-200 dark:bg-gray-700 sm:block" />
        <Button size="small" icon={<DesktopRegular />} onClick={() => onOpenBindings(row)}>
          {t("设备绑定")}
        </Button>
        <Button size="small" icon={<EditRegular />} onClick={() => onEdit(row)} />
        <Button size="small" variant="danger" icon={<DeleteRegular />} onClick={() => onDelete(row)} />
      </div>
    </div>
  );
}

export function UpstreamSection({ rows, loading, error, onRetry, onNew, onEdit, onDelete, onOpenBindings }: UpstreamSectionProps) {
  const { t } = useI18n();
  return (
    <div>
      {error ? (
        <ErrorState className="mb-6" title={t("加载上游代理失败")} message={error.message} statusCode={error.status} retryText={t("重试")} onRetry={onRetry} />
      ) : null}
      <div className="ui-card p-6">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-[#0ea5e9] to-[#0284c7] text-white shadow-lg shadow-indigo-500/25">
              <GlobeRegular className="text-[20px]" />
            </div>
            <div>
              <div className="text-lg font-bold text-gray-900 dark:text-white">{t("VoWiFi 上游代理")}</div>
              <div className="text-xs text-gray-500">{t("将设备的 VoWiFi 建链、IMS 和短信通信通过支持 UDP Associate 的 SOCKS5 代理传输。")}</div>
            </div>
          </div>
          <Button variant="primary" className="!border-0" icon={<AddRegular />} onClick={onNew}>
            {t("新增代理")}
          </Button>
        </div>
        {loading && rows.length === 0 ? (
          <ListSkeleton rows={2} />
        ) : rows.length === 0 ? (
          <EmptyState
            title={t("暂无上游代理")}
            subtitle={t("点击“新增代理”创建 SOCKS5 上游代理，然后将需要使用它的设备直接绑定；未绑定设备默认直连。")}
          />
        ) : (
          <div className="space-y-3">
            {rows.map((row) => (
              <UpstreamRowCard key={row.id} row={row} onEdit={onEdit} onDelete={onDelete} onOpenBindings={onOpenBindings} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
