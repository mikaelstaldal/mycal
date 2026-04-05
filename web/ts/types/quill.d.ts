declare class Quill {
  constructor(container: HTMLElement, options?: any);
  root: HTMLElement;
  getSelection(): { index: number; length: number } | null;
  formatText(index: number, length: number, format: string, value: any): void;
  on(event: string, handler: (...args: any[]) => void): this;
  getText(): string;
  setContents(delta: any): void;
}
