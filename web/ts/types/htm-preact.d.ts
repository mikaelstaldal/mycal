declare module 'htm/preact' {
  import type { VNode, ComponentChildren, ComponentType, Component } from 'preact';

  export function html(
    strings: TemplateStringsArray,
    ...values: any[]
  ): VNode | VNode[] | null;

  export { VNode, ComponentType, ComponentChildren, Component };
  export { h, render } from 'preact';
}
