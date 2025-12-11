/**
 * Wails v3 compatibility layer
 * Provides v2-style window.runtime and window.go.desktop.App API using v3 @wailsio/runtime
 */
import * as Events from '@wailsio/runtime';

// Import all App bindings
import * as App from '../../bindings/github.com/opik/miau/internal/desktop/app.js';

// Create v2-compatible runtime object
const runtime = {
  /**
   * Register an event listener (v2 style)
   * In v2: window.runtime.EventsOn(name, callback)
   * callback receives data directly, not WailsEvent object
   */
  EventsOn(eventName, callback) {
    return Events.Events.On(eventName, (ev) => {
      callback(ev.data);
    });
  },

  /**
   * Register an event listener that fires multiple times (v2 style)
   */
  EventsOnMultiple(eventName, callback, maxCallbacks) {
    return Events.Events.OnMultiple(eventName, (ev) => {
      callback(ev.data);
    }, maxCallbacks);
  },

  /**
   * Register a one-time event listener (v2 style)
   */
  EventsOnce(eventName, callback) {
    return Events.Events.Once(eventName, (ev) => {
      callback(ev.data);
    });
  },

  /**
   * Remove event listeners (v2 style)
   */
  EventsOff(eventName, ...additionalEventNames) {
    Events.Events.Off(eventName, ...additionalEventNames);
  },

  /**
   * Remove all event listeners (v2 style)
   */
  EventsOffAll() {
    Events.Events.OffAll();
  },

  /**
   * Emit an event (v2 style)
   * In v2: EventsEmit(name, ...data) where data is spread
   * In v3: Emit(name, data) where data is a single value
   */
  EventsEmit(eventName, ...data) {
    // v2 passes multiple args, v3 expects single data argument
    // If multiple args, pass as array; if single, pass directly
    const payload = data.length === 1 ? data[0] : data;
    return Events.Events.Emit(eventName, payload);
  }
};

// Expose on window for global access (v2 style)
if (typeof window !== 'undefined') {
  window.runtime = runtime;

  // Create v2-style window.go.desktop.App interface
  // This allows existing code using window.go.desktop.App.Method() to work
  window.go = window.go || {};
  window.go.desktop = window.go.desktop || {};
  window.go.desktop.App = App;
}

export default runtime;
export { App };
