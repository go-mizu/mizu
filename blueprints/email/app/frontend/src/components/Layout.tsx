import { useState } from "react";
import { Outlet, useNavigate } from "react-router-dom";
import { Menu, Settings as SettingsIcon } from "lucide-react";
import Sidebar from "./Sidebar";
import SearchBar from "./SearchBar";
import ComposeModal from "./ComposeModal";
import ToastContainer from "./Toast";
import { useEmailStore, useSettingsStore } from "../store";

export default function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const composeOpen = useEmailStore((s) => s.composeOpen);
  const settings = useSettingsStore((s) => s.settings);
  const navigate = useNavigate();

  const displayName = settings?.display_name ?? "User";
  const initial = displayName.charAt(0).toUpperCase();

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      {/* Header */}
      <header className="header-shadow relative z-30 flex h-16 flex-shrink-0 items-center bg-white px-2">
        <button
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className="flex h-12 w-12 items-center justify-center rounded-full hover:bg-gray-100"
          aria-label="Toggle sidebar"
        >
          <Menu className="h-5 w-5 text-gmail-text-secondary" />
        </button>

        <button
          onClick={() => navigate("/")}
          className="ml-1 flex items-center gap-2 px-2"
        >
          <svg
            viewBox="0 0 24 24"
            className="h-8 w-8"
            fill="none"
          >
            <path
              d="M1 5.5L12 13.5L23 5.5"
              stroke="#EA4335"
              strokeWidth="1.5"
              fill="none"
            />
            <path
              d="M1 5.5V18.5C1 19.05 1.45 19.5 2 19.5H22C22.55 19.5 23 19.05 23 18.5V5.5L12 13.5L1 5.5Z"
              fill="none"
              stroke="#EA4335"
              strokeWidth="0.5"
            />
            <rect
              x="1"
              y="4.5"
              width="22"
              height="15"
              rx="2"
              stroke="#EA4335"
              strokeWidth="1.2"
              fill="none"
            />
          </svg>
          <span
            className="text-xl"
            style={{ fontFamily: "'Google Sans', sans-serif", color: "#202124" }}
          >
            Mail
          </span>
        </button>

        <div className="mx-auto w-full max-w-2xl px-8">
          <SearchBar />
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={() => navigate("/settings")}
            className="flex h-10 w-10 items-center justify-center rounded-full hover:bg-gray-100"
            aria-label="Settings"
          >
            <SettingsIcon className="h-5 w-5 text-gmail-text-secondary" />
          </button>
          <div className="ml-2 flex h-8 w-8 items-center justify-center rounded-full bg-gmail-blue text-sm font-medium text-white cursor-pointer">
            {initial}
          </div>
        </div>
      </header>

      {/* Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <aside
          className={`flex-shrink-0 overflow-y-auto overflow-x-hidden bg-white transition-all duration-200 ${
            sidebarOpen ? "w-64" : "w-16"
          }`}
        >
          <Sidebar collapsed={!sidebarOpen} />
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto bg-gmail-bg">
          <div className="mx-2 mt-0 h-full rounded-t-2xl bg-white">
            <Outlet />
          </div>
        </main>
      </div>

      {/* Compose Modal */}
      {composeOpen && <ComposeModal />}

      {/* Toast Notifications */}
      <ToastContainer />
    </div>
  );
}
