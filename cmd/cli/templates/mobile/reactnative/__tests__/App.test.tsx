import React from 'react';
import renderer from 'react-test-renderer';
import LoadingView from '../src/components/LoadingView';
import ErrorView from '../src/components/ErrorView';

describe('LoadingView', () => {
  it('renders correctly without message', () => {
    const tree = renderer.create(<LoadingView />).toJSON();
    expect(tree).toMatchSnapshot();
  });

  it('renders correctly with message', () => {
    const tree = renderer.create(<LoadingView message="Loading..." />).toJSON();
    expect(tree).toMatchSnapshot();
  });
});

describe('ErrorView', () => {
  it('renders correctly with error', () => {
    const tree = renderer
      .create(<ErrorView error={new Error('Test error')} />)
      .toJSON();
    expect(tree).toMatchSnapshot();
  });

  it('renders correctly with retry button', () => {
    const tree = renderer
      .create(<ErrorView error={new Error('Test error')} onRetry={() => {}} />)
      .toJSON();
    expect(tree).toMatchSnapshot();
  });
});
