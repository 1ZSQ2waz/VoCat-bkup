import { useState } from "react";
import { cx } from "../lib/utils";
import { useI18n } from "../lib/i18n";
import { message } from "./ui/message";
import * as api from "../api";

const PHRASES = { zh: "我同意并确认", en: "I agree and confirm" } as const;

function WarningGlyph() {
  return (
    <svg className="h-8 w-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
        d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
      />
    </svg>
  );
}

function Item({ index, children }: { index: number; children: React.ReactNode }) {
  return (
    <div className="flex items-start">
      <div className="mr-3 mt-0.5 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-indigo-100 text-xs font-bold text-indigo-700 shadow-sm dark:bg-indigo-900/60 dark:text-indigo-300">
        {index}
      </div>
      <p>{children}</p>
    </div>
  );
}

// 中文条款（新增：仅限高通模块检测/测试卡、禁止 MCC 460、仅限美国硬件企业开发商）。
function ZhItems() {
  return (
    <>
      <Item index={1}>
        本软件（vocat）属于个人开发者业余时间开发的工具软件，支持高通模块调试，仅供技术研究、学习交流及企业内部测试使用。
        <strong className="text-indigo-600 dark:text-indigo-400">严禁用于任何形式的商业出售或转售</strong>
        ，严禁作为生产环境的基础设施。
      </Item>
      <Item index={2}>本项目仅用于高通模块功能正常检测使用，仅限接入测试类卡片使用。</Item>
      <Item index={3}>
        <strong className="text-red-500 dark:text-red-400">禁止 MCC 460 卡片进行测试。</strong>
      </Item>
      <Item index={4}>
        本软件面向使用配套设备进行高通模块调试的企业开发者提供。允许企业使用本软件对其设备进行调试，但
        <strong className="text-red-500 dark:text-red-400">严禁商业出售或转售</strong>
        ；如发现在此范围外的违规使用，
        <strong className="text-red-500 dark:text-red-400">我们将会自动锁定软件以及拉黑您的卡片 EID</strong>。
      </Item>
      <Item index={5}>
        使用者承诺将严格遵守所在国家或地区的相关法律法规。
        <strong className="text-red-500 dark:text-red-400">
          严禁将本软件用于电信诈骗、垃圾短信发送、非法网络代理、渗透测试等任何非法或违规场景
        </strong>
        。
      </Item>
      <Item index={6}>
        本软件涉及底层 Modem 通信操作，可能包含未知的缺陷。对于因使用本软件引发的硬件损坏、通信资费异常、隐私泄露等直接或间接损失，
        <strong>由使用者自行承担所有责任</strong>。
      </Item>
      <Item index={7}>
        一旦点击继续即表示无条件接受本协议。如果您拒绝，本软件将立即触发自毁与环境清理机制以确保设备安全。
      </Item>
    </>
  );
}

// English clauses (mirror of the Chinese items).
function EnItems() {
  return (
    <>
      <Item index={1}>
        This software (vocat) is a utility built by an independent developer in their spare time. It supports Qualcomm
        module debugging and is provided only for technical research, learning, and enterprise internal testing.{" "}
        <strong className="text-indigo-600 dark:text-indigo-400">
          It is strictly prohibited to sell or resell it commercially in any form
        </strong>{" "}
        or to use it as production infrastructure.
      </Item>
      <Item index={2}>
        This project is intended solely for verifying the proper functioning of Qualcomm modules; only test-class
        SIM cards may be connected.
      </Item>
      <Item index={3}>
        <strong className="text-red-500 dark:text-red-400">
          Testing with MCC 460 (China) SIM cards is strictly prohibited.
        </strong>
      </Item>
      <Item index={4}>
        This software is provided to enterprise developers who use supporting equipment to debug Qualcomm modules.
        Enterprises may use this software to debug their own equipment, but{" "}
        <strong className="text-red-500 dark:text-red-400">
          commercial sale or resale is strictly prohibited
        </strong>
        ; if misuse outside this scope is detected,{" "}
        <strong className="text-red-500 dark:text-red-400">
          the software will be automatically locked and your card&apos;s EID will be blacklisted
        </strong>
        .
      </Item>
      <Item index={5}>
        The user undertakes to strictly comply with the laws and regulations of their country or region.{" "}
        <strong className="text-red-500 dark:text-red-400">
          It is strictly prohibited to use this software for telecom fraud, spam messaging, illegal network
          proxying, penetration testing, or any other illegal or non-compliant scenario
        </strong>
        .
      </Item>
      <Item index={6}>
        This software involves low-level modem communication and may contain unknown defects.{" "}
        <strong>The user bears all responsibility</strong> for any direct or indirect losses arising from its use,
        including hardware damage, abnormal carrier charges, or privacy leakage.
      </Item>
      <Item index={7}>
        Clicking continue constitutes unconditional acceptance of this agreement. If you decline, the software
        will immediately trigger its self-destruct and environment cleanup mechanism to keep the device safe.
      </Item>
    </>
  );
}

const OVERLAY_STYLE =
  "display:flex;height:100vh;background:#0a0a0a;align-items:center;justify-content:center;" +
  "font-size:24px;color:#ef4444;font-weight:bold;font-family:sans-serif;flex-direction:column;gap:16px;";
const OVERLAY_ICON =
  '<svg style="width:64px;height:64px;" fill="none" viewBox="0 0 24 24" stroke="currentColor">' +
  '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" ' +
  'd="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>';

// Disclaimer / EULA overlay shown after login (first run requires typing the
// phrase; subsequent periodic confirmations only require a click).
export function Disclaimer({
  firstTime,
  onAgree,
}: {
  firstTime: boolean;
  onAgree: () => void;
}) {
  const { t, lang } = useI18n();
  const zh = lang === "zh";
  const phrase = PHRASES[lang];
  const [typed, setTyped] = useState("");
  const canAgree = !firstTime || typed === phrase;

  function reject() {
    message.warning(zh ? t("正在退出并清理软件...") : "Exiting and cleaning up...");
    api
      .api("/system/uninstall", { method: "POST" })
      .catch(() => {})
      .finally(() => {
        const text = zh ? t("软件已被卸载 / 服务已终止") : "Software uninstalled / service stopped";
        document.body.innerHTML = `<div style="${OVERLAY_STYLE}"><div>${OVERLAY_ICON}</div><div>${text}</div></div>`;
      });
  }

  return (
    <div className="disclaimer-overlay fixed inset-0 z-[9999] flex items-center justify-center bg-black/60 backdrop-blur-md">
      <div className="disclaimer-dialog relative mx-4 w-full max-w-lg overflow-hidden rounded-3xl border border-white/20 bg-white/90 p-8 shadow-2xl backdrop-blur-2xl dark:border-gray-700/50 dark:bg-gray-900/90">
        <div className="pointer-events-none absolute left-0 top-0 h-32 w-full bg-gradient-to-b from-indigo-500/20 to-transparent" />
        <div className="disclaimer-dialog-content relative z-10 flex min-h-0 flex-col">
          <div className="disclaimer-icon mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-[#0ea5e9] to-[#0284c7] shadow-lg shadow-indigo-500/30">
            <WarningGlyph />
          </div>
          <h2 className="disclaimer-title mb-5 text-center text-2xl font-extrabold tracking-tight text-gray-900 dark:text-white">
            {zh ? t("vocat 最终用户许可与免责声明") : "vocat End User License Agreement & Disclaimer"}
          </h2>
          <div className="disclaimer-body space-y-4 text-[14px] font-medium leading-relaxed text-gray-600 dark:text-gray-300">
            {zh ? <ZhItems /> : <EnItems />}
          </div>
          <div className="disclaimer-actions mt-6 border-t border-gray-100 pt-5 dark:border-gray-800">
            {firstTime ? (
              <p className="mb-3 text-center text-xs font-bold text-gray-500 dark:text-gray-400">
                {zh ? t("请输入") : "Please type"}「
                <span className="select-all text-indigo-600 dark:text-indigo-400">{phrase}</span>」
                {zh ? t("以解锁按钮") : "to unlock the button"}
              </p>
            ) : (
              <p className="mb-3 text-center text-xs font-bold text-gray-500 dark:text-gray-400">
                {zh ? t("本次为周期性确认，点击") : "Periodic confirmation. Click"}「
                <span className="text-indigo-600 dark:text-indigo-400">{phrase}</span>」
                {zh ? t("即可继续") : "to continue"}
              </p>
            )}
            {firstTime && (
              <div className="disclaimer-input-wrap mb-5">
                <input
                  type="text"
                  value={typed}
                  onChange={(event) => setTyped(event.target.value)}
                  onPaste={(event) => event.preventDefault()}
                  autoComplete="off"
                  placeholder={zh ? `请输入：${phrase}` : `Please type: ${phrase}`}
                  className="w-full rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 text-center text-sm font-semibold outline-none transition-all placeholder-gray-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/50 dark:border-gray-700 dark:bg-gray-800/80 dark:text-white dark:placeholder-gray-500 dark:focus:border-indigo-500"
                />
              </div>
            )}
            <div className="disclaimer-button-row flex gap-4">
              <button
                type="button"
                onClick={reject}
                className="flex-1 rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm font-bold tracking-wide text-gray-500 transition-all duration-300 hover:border-red-200 hover:bg-red-50 hover:text-red-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:border-red-900/50 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                {zh ? t("拒绝并卸载") : "Decline & Uninstall"}
              </button>
              <button
                type="button"
                onClick={() => canAgree && onAgree()}
                disabled={!canAgree}
                className={cx(
                  "flex-[1.5] rounded-xl px-4 py-3 text-sm font-bold tracking-wide transition-all duration-300",
                  canAgree
                    ? "cursor-pointer bg-gradient-to-r from-[#0ea5e9] to-[#0284c7] text-white shadow-lg shadow-indigo-500/30 hover:-translate-y-0.5 hover:shadow-indigo-500/50 active:translate-y-0"
                    : "cursor-not-allowed border border-gray-300 bg-gray-200 text-gray-400 opacity-60 shadow-none dark:border-gray-700 dark:bg-gray-800 dark:text-gray-500",
                )}
              >
                {firstTime ? (zh ? t("同意并继续") : "Agree & Continue") : phrase}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
