import './OfflineScreen.css';

export default function OfflineScreen() {
  const handleRetry = () => {
    window.location.reload();
  };

  return (
    <div className="offline-screen-container">
      <div className="offline-screen-content">
        <div className="offline-screen-icon-container">
          <span className="offline-screen-icon">&#128268;</span>
        </div>
        <h1 className="offline-screen-title">You're Offline</h1>
        <p className="offline-screen-subtitle">
          Please check your internet connection and try again.
        </p>
        <button className="offline-screen-button" onClick={handleRetry}>
          Retry
        </button>
      </div>
    </div>
  );
}
