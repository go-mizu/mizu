import { describe, test, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import LoadingView from '../src/components/LoadingView';
import ErrorView from '../src/components/ErrorView';

describe('LoadingView', () => {
  test('renders without message', () => {
    render(<LoadingView />);
    expect(document.querySelector('.loading-spinner')).toBeTruthy();
  });

  test('renders with message', () => {
    render(<LoadingView message="Loading..." />);
    expect(screen.getByText('Loading...')).toBeTruthy();
  });
});

describe('ErrorView', () => {
  test('renders error message', () => {
    render(<ErrorView error={new Error('Test error')} />);
    expect(screen.getByText('Test error')).toBeTruthy();
  });

  test('renders retry button when onRetry provided', () => {
    render(<ErrorView error={new Error('Test error')} onRetry={() => {}} />);
    expect(screen.getByText('Try Again')).toBeTruthy();
  });
});
