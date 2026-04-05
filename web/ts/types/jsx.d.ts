declare namespace JSX {
  type Element = import('preact').VNode;
  interface IntrinsicElements {
    [elemName: string]: any;
  }
  interface ElementClass {
    render(): any;
  }
}
