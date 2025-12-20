import './OfflineBanner.css';

export default function OfflineBanner() {
  return (
    <div className="offline-banner">
      <span className="offline-icon">&#9888;</span>
      <span>You are currently offline</span>
    </div>
  );
}
