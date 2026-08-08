import { useI18n } from "../../lib/i18n";
import { BrandLogo } from "../shell/BrandLogo";
// LoadingScreen: glass card with brand mark, title/subtitle, shimmer bar and spinner.
export function LoadingScreen({
  title,
  subtitle,
}: {
  title?: string;
  subtitle?: string;
}) {
  const { t } = useI18n();
  const shownTitle = title ?? t("正在加载…");
  const shownSubtitle = subtitle ?? t("首次打开会稍慢，资源缓存后会更快");
  return (
    <div className="flex min-h-[260px] w-full items-center justify-center">
      <div className="w-full max-w-md rounded-2xl border border-white/10 bg-white/5 p-6 shadow-2xl backdrop-blur-xl dark:bg-white/5">
        <div className="flex items-center gap-3">
          <BrandLogo className="h-11 w-11 flex-shrink-0" />
          <div className="min-w-0">
            <div className="truncate text-base font-extrabold text-gray-900 dark:text-white">{shownTitle}</div>
            <div className="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{shownSubtitle}</div>
          </div>
        </div>
        <div className="mt-5 flex items-center gap-3">
          <div className="text-xs font-semibold tracking-wide text-gray-500 dark:text-gray-400">loading</div>
          <div className="relative h-2 flex-1 overflow-hidden rounded-full bg-gray-200/50 dark:bg-white/10">
            <div className="absolute inset-0 -translate-x-[60%] animate-[loader-shimmer_1.05s_ease-in-out_infinite] bg-gradient-to-r from-transparent via-white/35 to-transparent" />
          </div>
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-white/25 border-t-white" />
        </div>
      </div>
    </div>
  );
}
