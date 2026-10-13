import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

export default function CodeBlock({ children, label }: { children: string; label?: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard
      ?.writeText(children)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      })
      .catch(() => {});
  };

  return (
    <div className="rounded-lg bg-code text-code-ink overflow-hidden">
      <div className="flex items-center justify-between gap-3 px-4 py-2 border-b border-white/10 text-xs text-code-muted">
        <span className="font-mono truncate">{label ?? "shell"}</span>
        <button
          type="button"
          onClick={copy}
          className="flex items-center gap-1.5 hover:text-code-ink transition-colors"
          aria-label={copied ? t("common.copied") : t("common.copy")}
        >
          {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          <span aria-hidden="true">{copied ? t("common.copied") : t("common.copy")}</span>
        </button>
      </div>
      <pre className="p-4 font-mono text-[12px] sm:text-[13px] leading-relaxed overflow-x-auto">
        <code className="whitespace-pre">{children}</code>
      </pre>
    </div>
  );
}
