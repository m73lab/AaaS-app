type Handler = (event: string, payload: unknown) => void;

export class EventBus {
  private handlers = new Map<string, Handler[]>();

  on(event: string, handler: Handler): void {
    const list = this.handlers.get(event) ?? [];
    list.push(handler);
    this.handlers.set(event, list);
  }

  emit(event: string, payload: unknown): void {
    const list = this.handlers.get(event) ?? [];
    for (const h of list) {
      try {
        h(event, payload);
      } catch (err) {
        console.error(`[EventBus] handler error for ${event}:`, err);
      }
    }
  }
}
