import { ServiceKey } from './service-keys.js';

type Factory = (container: Container) => unknown;

export class Container {
  private registry = new Map<ServiceKey, { factory: Factory; singleton: boolean; instance?: unknown }>();

  register(key: ServiceKey, factory: Factory, opts: { singleton?: boolean } = {}): this {
    this.registry.set(key, { factory, singleton: opts.singleton ?? true });
    return this;
  }

  get<T = unknown>(key: ServiceKey): T {
    const entry = this.registry.get(key);
    if (!entry) throw new Error(`Service not registered: ${String(key)}`);
    if (entry.singleton) {
      if (entry.instance === undefined) entry.instance = entry.factory(this);
      return entry.instance as T;
    }
    return entry.factory(this) as T;
  }

  has(key: ServiceKey): boolean {
    return this.registry.has(key);
  }
}
