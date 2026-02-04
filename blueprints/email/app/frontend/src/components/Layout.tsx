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
    <div className="flex h-screen flex-col overflow-hidden bg-gmail-bg">
      {/* ---- Fixed Header (64px) ---- */}
      <header className="header-shadow relative z-30 flex h-16 flex-shrink-0 items-center bg-white px-2">
        {/* Left: Hamburger + Logo */}
        <div className="flex items-center">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="flex h-12 w-12 items-center justify-center rounded-full hover:bg-gmail-surface-variant focus-ring"
            aria-label="Toggle sidebar"
          >
            <Menu className="h-5 w-5 text-gmail-text-secondary" />
          </button>

          <button
            onClick={() => navigate("/")}
            className="ml-1 flex items-center gap-2.5 rounded-lg px-3 py-1.5 hover:bg-gmail-surface-container focus-ring"
          >
            <svg
              viewBox="0 0 24 24"
              className="h-8 w-8"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              {/* Envelope body */}
              <rect
                x="1.5"
                y="4.5"
                width="21"
                height="15"
                rx="2"
                stroke="#EA4335"
                strokeWidth="1.2"
                fill="none"
              />
              {/* Flap lines - multicolor Gmail style */}
              <path
                d="M1.5 6.5L12 14L22.5 6.5"
                stroke="#EA4335"
                strokeWidth="1.3"
                strokeLinecap="round"
                strokeLinejoin="round"
                fill="none"
              />
              {/* Left accent */}
              <path
                d="M1.5 6.5L1.5 18.5"
                stroke="#4285F4"
                strokeWidth="1.2"
                strokeLinecap="round"
                fill="none"
              />
              {/* Right accent */}
              <path
                d="M22.5 6.5L22.5 18.5"
                stroke="#34A853"
                strokeWidth="1.2"
                strokeLinecap="round"
                fill="none"
              />
              {/* Bottom accent */}
              <path
                d="M1.5 19.5L22.5 19.5"
                stroke="#FBBC04"
                strokeWidth="1.2"
                strokeLinecap="round"
                fill="none"
              />
            </svg>
            <span className="font-google-sans text-[22px] font-normal text-gmail-text-primary">
              Mail
            </span>
          </button>
        </div>

        {/* Center: Search Bar */}
        <div className="mx-auto w-full max-w-2xl px-8">
          <SearchBar />
        </div>

        {/* Right: Settings + Avatar */}
        <div className="flex items-center gap-1">
          <button
            onClick={() => navigate("/settings")}
            className="flex h-10 w-10 items-center justify-center rounded-full hover:bg-gmail-surface-variant focus-ring"
            aria-label="Settings"
          >
            <SettingsIcon className="h-5 w-5 text-gmail-text-secondary" />
          </button>
          <div
            className="ml-2 flex h-9 w-9 cursor-pointer items-center justify-center rounded-full bg-gmail-blue text-sm font-medium text-white ring-2 ring-transparent transition-all hover:ring-gmail-blue-light"
            title={displayName}
          >
            {initial}
          </div>
        </div>
      </header>

      {/* ---- Body: Sidebar + Main ---- */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <aside
          className="flex-shrink-0 overflow-y-auto overflow-x-hidden bg-white transition-[width] duration-200 ease-in-out"
          style={{ width: sidebarOpen ? "var(--gmail-sidebar-width)" : "var(--gmail-sidebar-collapsed-width)" }}
        >
          <Sidebar collapsed={!sidebarOpen} />
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 overflow-y-auto bg-gmail-bg">
          <div className="mx-2 mt-0 h-full rounded-t-2xl bg-white shadow-sm">
            <Outlet />
          </div>
        </main>
      </div>

      {/* ---- Compose Modal ---- */}
      {composeOpen && <ComposeModal />}

      {/* ---- Toast Notifications ---- */}
      <ToastContainer />
    </div>
  );
}
