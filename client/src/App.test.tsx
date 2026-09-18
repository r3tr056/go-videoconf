import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import App from './App';

describe('App graphite UI', () => {
  it('renders the home screen with join and create actions', async () => {
    render(<App />);

    expect(
      await screen.findByRole('heading', { name: /video calls, self-hosted/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/meeting code/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /join meeting/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /new meeting/i })).toBeInTheDocument();
  });

  it('opens the create-meeting dialog', async () => {
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: /^new meeting$/i }));

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/meeting title/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/meeting password/i)).toBeInTheDocument();
  });

  it('exposes join options as switches', async () => {
    render(<App />);

    const camera = await screen.findByRole('switch', { name: /camera on when joining/i });
    const mic = await screen.findByRole('switch', { name: /microphone on when joining/i });
    expect(camera).toHaveAttribute('data-state', 'checked');
    expect(mic).toHaveAttribute('data-state', 'checked');

    fireEvent.click(camera);
    await waitFor(() => {
      expect(camera).toHaveAttribute('data-state', 'unchecked');
    });
  });
});
