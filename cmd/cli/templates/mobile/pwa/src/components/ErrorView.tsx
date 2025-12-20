import { MizuError } from '../runtime/errors';
import './ErrorView.css';

interface ErrorViewProps {
  error: Error | MizuError;
  onRetry?: () => void;
}

export default function ErrorView({ error, onRetry }: ErrorViewProps) {
  const message = error instanceof MizuError ? error.message : error.message;

  return (
    <div className="error-container">
      <div className="error-icon">!</div>
      <h2 className="error-title">Something went wrong</h2>
      <p className="error-message">{message}</p>
      {onRetry && (
        <button className="error-button" onClick={onRetry}>
          Try Again
        </button>
      )}
    </div>
  );
}
