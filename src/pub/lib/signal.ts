module Q {

/**
 * An event dispatcher allowing clients to subscribe (tap), unsubscribe (untap) and
 * dispatch (raise) events.
 */
export class Signal {
  // All callbacks that have tapped this signal
  listeners : { (...args : any[]) : void; } [] = [];

  /**
   * Listen for this signal to be raised.
   * @param l the callback for the listener
   */
  tap(l : (...arg : any[]) => void) : void {
    // Make a copy of the listeners to avoid the all too common
    // subscribe-during-dispatch problem
    this.listeners = this.listeners.slice(0);
    this.listeners.push(l);
  }

  /**
   * Stop listening for this signal to be raised.
   * @param l the callback to be removed as a listener
   */
  untap(l : (...arg : any[]) => void) : void {
    var ix = this.listeners.indexOf(l);
    if (ix == -1) {
      return;
    }

    // Make a copy of the listeners to avoid the all to common
    // unsubscribe-during-dispatch problem
    this.listeners = this.listeners.slice(0);
    this.listeners.splice(ix, 1);
  }

  /**
   * Raise the signal for all listeners and pass allowing the given arguments.
   * @param args an arbitrary list of arguments to be passed to listeners
   */
  raise(...args : any[]) : void {
    this.listeners.forEach((l) => {
      l.apply(this, args);
    });
  }
}

}