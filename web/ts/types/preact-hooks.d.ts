declare module 'preact/hooks' {
  import type { RefObject } from 'preact';

  export type Dispatch<A> = (value: A) => void;
  export type SetStateAction<S> = S | ((prevState: S) => S);
  export type Reducer<S, A> = (prevState: S, action: A) => S;
  export type EffectCallback = () => void | (() => void | undefined);
  export type DependencyList = ReadonlyArray<unknown>;

  export function useState<S>(initialState: S | (() => S)): [S, Dispatch<SetStateAction<S>>];
  export function useState<S = undefined>(): [S | undefined, Dispatch<SetStateAction<S | undefined>>];

  export function useEffect(effect: EffectCallback, deps?: DependencyList): void;
  export function useLayoutEffect(effect: EffectCallback, deps?: DependencyList): void;

  export function useRef<T>(initialValue: T): RefObject<T>;
  export function useRef<T>(initialValue: T | null): RefObject<T | null>;
  export function useRef<T = undefined>(): RefObject<T | undefined>;

  export function useCallback<T extends (...args: any[]) => any>(
    callback: T,
    deps: DependencyList
  ): T;

  export function useMemo<T>(factory: () => T, deps: DependencyList | undefined): T;

  export function useReducer<S, A>(
    reducer: Reducer<S, A>,
    initialState: S
  ): [S, Dispatch<A>];

  export function useContext<T>(context: any): T;
}
