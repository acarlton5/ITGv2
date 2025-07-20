// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom';

// Provide a basic MediaStream stub for tests
global.MediaStream = class {
  addTrack() {}
};

// Stub RTCPeerConnection for tests
global.RTCPeerConnection = class {
  addIceCandidate() {}
  createAnswer() { return {}; }
  setLocalDescription() {}
  setRemoteDescription() {}
  addEventListener() {}
};

// Simple WebSocket mock
global.WebSocket = class {
  constructor(url) {
    this.url = url;
    this.readyState = 1; // OPEN
  }
  send() {}
  close() { this.readyState = 3; }
  addEventListener() {}
  removeEventListener() {}
};

// Mock fetch used by SocketProvider
global.fetch = jest.fn(() =>
  Promise.resolve({ json: () => Promise.resolve({ wsUrl: 'ws://localhost' }) })
);
