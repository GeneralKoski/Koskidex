import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RefreshCcw, Home } from "lucide-react";
import { withTranslation, type WithTranslation } from "react-i18next";

interface Props extends WithTranslation {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
  };

  public static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Uncaught error:", error, errorInfo);
  }

  public render() {
    const { t } = this.props;
    if (this.state.hasError) {
      return (
        <div className="min-h-screen bg-paper text-ink flex items-center px-4 md:px-6">
          <div className="max-w-xl mx-auto">
            <AlertTriangle className="w-8 h-8 text-bad" aria-hidden="true" />
            <h1 className="mt-4 font-wide font-black text-4xl md:text-5xl tracking-[-0.03em]">{t("error.oops")}</h1>
            <p className="mt-4 text-lg text-muted leading-relaxed">
              {t("error.something_wrong")}
            </p>
            <div className="mt-8 flex flex-col sm:flex-row gap-3">
              <button
                type="button"
                onClick={() => window.location.reload()}
                className="inline-flex items-center justify-center gap-2 px-5 py-3 rounded-md bg-ink text-paper font-semibold hover:bg-link transition-colors"
              >
                <RefreshCcw className="w-4 h-4" />
                {t("error.retry")}
              </button>
              <a
                href="/"
                className="inline-flex items-center justify-center gap-2 px-5 py-3 rounded-md border border-rule font-semibold hover:border-ink transition-colors"
              >
                <Home className="w-4 h-4" />
                {t("docs.back_home")}
              </a>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

const ErrorBoundaryWithTranslation = withTranslation()(ErrorBoundary);
export default ErrorBoundaryWithTranslation;
