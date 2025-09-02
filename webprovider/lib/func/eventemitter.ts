export class EventEmitter<T> {
  private listeners: ((message: T) => void)[] = [];

  on(listener: (message: T) => void): void {
    this.listeners.push(listener);
  }

  emit(message: T): void {
    this.listeners.forEach((listener) => listener(message));
  }

  off(listener: (message: T) => void): void {
    const index = this.listeners.indexOf(listener);
    if (index !== -1) {
      this.listeners.splice(index, 1);
    }
  }
}
