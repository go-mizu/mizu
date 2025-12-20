import './LoadingView.css';

interface LoadingViewProps {
  message?: string;
}

export default function LoadingView({ message }: LoadingViewProps) {
  return (
    <div className="loading-container">
      <div className="loading-spinner" />
      {message && <p className="loading-message">{message}</p>}
    </div>
  );
}
