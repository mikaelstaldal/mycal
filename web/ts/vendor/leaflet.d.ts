declare namespace L {
  interface Map {
    setView(latlng: [number, number], zoom: number): this;
    remove(): void;
    on(event: string, handler: (...args: any[]) => void): this;
    invalidateSize(): void;
    panTo(latlng: [number, number] | { lat: number; lng: number }): void;
  }
  interface TileLayer {
    addTo(map: Map): this;
  }
  interface Marker {
    addTo(map: Map): this;
    setLatLng(latlng: [number, number] | { lat: number; lng: number }): this;
    getLatLng(): { lat: number; lng: number };
    on(event: string, handler: (...args: any[]) => void): this;
  }
  function map(el: HTMLElement, options?: any): Map;
  function tileLayer(url: string, options?: any): TileLayer;
  function marker(latlng: [number, number] | { lat: number; lng: number }, options?: any): Marker;
  function latLng(lat: number, lng: number): { lat: number; lng: number };
}
