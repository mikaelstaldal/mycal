declare module 'preact' {
  export interface VNode<P = {}> {
    type: string | ComponentType<any>;
    props: P & { children?: ComponentChildren };
    key: string | number | null;
  }

  export type ComponentChildren =
    | VNode
    | string
    | number
    | boolean
    | null
    | undefined
    | ComponentChildren[];

  export type ComponentType<P = {}> = FunctionComponent<P>;

  export interface FunctionComponent<P = {}> {
    (props: P): VNode | null;
  }

  export interface RefObject<T> {
    current: T | null;
  }

  export function h(
    type: string | ComponentType<any>,
    props: Record<string, any> | null,
    ...children: ComponentChildren[]
  ): VNode;

  export function render(vnode: VNode | null, parent: Element | null): void;

  export function createRef<T = any>(): RefObject<T>;

  export class Component<P = {}, S = {}> {
    props: P;
    state: S;
    setState(state: Partial<S> | ((prev: S) => Partial<S>)): void;
    render(): VNode | null;
  }
}
