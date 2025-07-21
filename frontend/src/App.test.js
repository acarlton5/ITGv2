import { render, screen } from '@testing-library/react';
import App from './App';
import RTCProvider from './context/RTCPeerContext';
import SocketProvider from './context/SocketContext';

test('renders Project ITG header', () => {
  render(
    <RTCProvider>
      <SocketProvider>
        <App />
      </SocketProvider>
    </RTCProvider>
  );
  const headers = screen.getAllByText(/Project ITG/i);
  expect(headers.length).toBeGreaterThan(0);
});
