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
        <div className="min-h-screen bg-[#0B1120] flex items-center justify-center p-6 text-center">
          <div className="glass-effect p-12 max-w-lg rounded-[2.5rem] border-red-500/20 shadow-2xl shadow-red-500/10">
            <div className="w-20 h-20 bg-red-500/10 rounded-3xl flex items-center justify-center mx-auto mb-8 animate-pulse">
              <AlertTriangle className="w-10 h-10 text-red-500" />
            </div>
            <h1 className="text-4xl font-black text-white mb-4 tracking-tight uppercase">{t("error.oops")}</h1>
            <p className="text-slate-400 text-lg mb-10 leading-relaxed font-light">
              {t("error.something_wrong")}
            </p>
            <div className="flex flex-col gap-4">
              <button
                onClick={() => window.location.reload()}
                className="btn btn-primary py-4 rounded-2xl font-black flex items-center justify-center gap-2 transform hover:scale-105 transition-transform"
              >
                <RefreshCcw className="w-5 h-5" />
                {t("error.retry")}
              </button>
              <a
                href="/"
                className="text-slate-500 hover:text-white text-sm font-bold flex items-center justify-center gap-2 transition-colors mt-4"
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

export default withTranslation()(ErrorBoundary);
