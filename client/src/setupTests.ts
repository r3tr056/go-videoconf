import '@testing-library/jest-dom';

// jsdom lacks several browser APIs that Radix primitives touch at render
// time. Minimal faithful-enough shims for the unit suite.
class ResizeObserverMock {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}

global.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver;

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

window.HTMLElement.prototype.scrollIntoView = function scrollIntoView(): void {};
