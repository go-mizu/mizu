import { useEffect } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { useLabelStore, useSettingsStore } from "./store";
import Layout from "./components/Layout";
import MailPage from "./pages/MailPage";
import SettingsPage from "./pages/SettingsPage";

export default function App() {
  const fetchLabels = useLabelStore((s) => s.fetchLabels);
  const fetchSettings = useSettingsStore((s) => s.fetchSettings);

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
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
