import { render, screen } from '@testing-library/react';
import App from './App';
import RTCProvider from './context/RTCPeerContext';
import SocketProvider from './context/SocketContext';

test('renders Project Lightspeed header', () => {
  render(
    <RTCProvider>
      <SocketProvider>
        <App />
      </SocketProvider>
    </RTCProvider>
  );
  const headers = screen.getAllByText(/Project Lightspeed/i);
  expect(headers.length).toBeGreaterThan(0);
});
