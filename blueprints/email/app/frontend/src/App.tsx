import { useEffect, lazy, Suspense } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { useLabelStore, useSettingsStore } from "./store";
import Layout from "./components/Layout";
import MailPage from "./pages/MailPage";

const SettingsPage = lazy(() => import("./pages/SettingsPage"));

export default function App() {
  const fetchLabels = useLabelStore((s) => s.fetchLabels);
  const fetchSettings = useSettingsStore((s) => s.fetchSettings);

  // Fetch labels and settings on mount
  useEffect(() => {
    fetchLabels();
    fetchSettings();
  }, [fetchLabels, fetchSettings]);

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<MailPage />} />
        <Route path="/label/:labelId" element={<MailPage />} />
        <Route path="/email/:emailId" element={<MailPage />} />
        <Route path="/settings" element={<Suspense><SettingsPage /></Suspense>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
