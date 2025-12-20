// API Response Types
export interface IPInfo {
  ip: string;
  hostname?: string;
  country?: string;
  iso_code?: string;
  in_eu?: boolean;
  city?: string;
  region?: string;
  latitude?: number;
  longitude?: number;
  timezone?: string;
  asn?: number;
  organization?: string;
  attribution: string;
}

export interface ErrorResponse {
  error: string;
  attribution: string;
}

export interface FeaturesResponse {
  onlineFeatures: boolean;
}

// Feature flags state
let featureFlags: FeaturesResponse = {
  onlineFeatures: false,
};

// Leaflet types (loaded globally via CDN)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
declare const L: any;

// Map instance
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let map: any = null;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let marker: any = null;

// API Client
export async function lookupIP(ip?: string): Promise<IPInfo> {
  const url = ip ? `/api/ip?ip=${encodeURIComponent(ip)}` : '/api/ip';
  
  const response = await fetch(url);
  
  if (!response.ok) {
    const errorData: ErrorResponse = await response.json();
    throw new Error(errorData.error || 'Failed to lookup IP');
  }
  
  return response.json();
}

export async function fetchFeatures(): Promise<FeaturesResponse> {
  try {
    const response = await fetch('/api/features');
    if (response.ok) {
      return response.json();
    }
  } catch (error) {
    console.error('Failed to fetch features:', error);
  }
  // Return defaults if fetch fails
  return { onlineFeatures: false };
}

// DOM Helpers
function getElementById<T extends HTMLElement>(id: string): T | null {
  return document.getElementById(id) as T | null;
}

function setText(id: string, value: string | number | boolean | undefined | null): void {
  const element = getElementById(id);
  if (element) {
    if (value === undefined || value === null || value === '') {
      element.textContent = '-';
    } else if (typeof value === 'boolean') {
      element.textContent = value ? 'Yes' : 'No';
    } else {
      element.textContent = String(value);
    }
  }
}

function showElement(id: string): void {
  const element = getElementById(id);
  if (element) {
    element.classList.remove('hidden');
  }
}

function hideElement(id: string): void {
  const element = getElementById(id);
  if (element) {
    element.classList.add('hidden');
  }
}

// UI State Management
function showLoading(): void {
  hideElement('results');
  hideElement('error');
  showElement('loading');
}

// Initialize or update the map
function updateMap(latitude: number, longitude: number, city?: string, country?: string): void {
  const mapContainer = document.getElementById('map');
  if (!mapContainer) return;

  // Create popup content
  const popupContent = [city, country].filter(Boolean).join(', ') || 'IP Location';

  if (!map) {
    // Initialize map
    map = L.map('map').setView([latitude, longitude], 10);
    
    // Add OpenStreetMap tiles
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© OpenStreetMap contributors',
      maxZoom: 18,
    }).addTo(map);

    // Add marker
    marker = L.marker([latitude, longitude])
      .addTo(map)
      .bindPopup(popupContent)
      .openPopup();
  } else {
    // Update existing map
    map.setView([latitude, longitude], 10);
    
    if (marker) {
      marker.setLatLng([latitude, longitude]);
      marker.setPopupContent(popupContent);
      marker.openPopup();
    }
  }

  // Force map to recalculate size (needed when container was hidden)
  setTimeout(() => {
    map?.invalidateSize();
  }, 100);
}

function showResults(data: IPInfo, isOwnIP: boolean): void {
  hideElement('loading');
  hideElement('error');
  
  // Update header label based on whether it's user's own IP or searched IP
  setText('result-label', isOwnIP ? 'Your IP Address' : 'IP Address Information');
  
  // Update all result fields
  setText('result-ip', data.ip);
  setText('result-country', data.country);
  setText('result-iso-code', data.iso_code);
  setText('result-in-eu', data.in_eu);
  setText('result-city', data.city);
  setText('result-region', data.region);
  
  // Combined coordinates display
  if (data.latitude !== undefined && data.longitude !== undefined) {
    setText('result-coordinates', `${data.latitude.toFixed(4)}, ${data.longitude.toFixed(4)}`);
  } else {
    setText('result-coordinates', undefined);
  }
  
  setText('result-timezone', data.timezone);
  setText('result-asn', data.asn ? `AS${data.asn}` : undefined);
  setText('result-organization', data.organization);
  
  // Only show hostname if online features are enabled
  if (featureFlags.onlineFeatures) {
    setText('result-hostname', data.hostname);
    showElement('hostname-row');
  } else {
    hideElement('hostname-row');
  }
  
  showElement('results');

  // Update map if coordinates are available
  if (data.latitude !== undefined && data.longitude !== undefined) {
    updateMap(data.latitude, data.longitude, data.city, data.country);
  }

  // Set up copy functionality for IP address
  setupCopyButton(data.ip);
}

function showError(message: string): void {
  hideElement('loading');
  hideElement('results');
  
  setText('error-message', message);
  showElement('error');
}

// Copy IP functionality
function setupCopyButton(ip: string): void {
  const copyBtn = getElementById<HTMLButtonElement>('copy-ip-btn');
  if (!copyBtn) return;

  // Remove any existing click listeners by cloning the button
  const newBtn = copyBtn.cloneNode(true) as HTMLButtonElement;
  copyBtn.parentNode?.replaceChild(newBtn, copyBtn);

  newBtn.addEventListener('click', async (e) => {
    e.preventDefault();
    
    try {
      await navigator.clipboard.writeText(ip);
      
      // Visual feedback: change icon to checkmark temporarily
      const originalHTML = newBtn.innerHTML;
      newBtn.innerHTML = `
        <svg class="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
        </svg>
      `;
      
      // Reset after 1.5 seconds
      setTimeout(() => {
        newBtn.innerHTML = originalHTML;
      }, 1500);
    } catch (error) {
      console.error('Failed to copy IP address:', error);
      // Optional: show error feedback
    }
  });
}

// Event Handlers
async function handleLookup(ip?: string): Promise<void> {
  showLoading();
  
  const isOwnIP = !ip;
  
  // Update URL with IP parameter (without page reload)
  if (ip) {
    const url = new URL(window.location.href);
    url.searchParams.set('ip', ip);
    window.history.pushState({}, '', url.toString());
  } else {
    // Clear the IP parameter if looking up own IP
    const url = new URL(window.location.href);
    url.searchParams.delete('ip');
    window.history.pushState({}, '', url.toString());
  }
  
  try {
    const data = await lookupIP(ip);
    showResults(data, isOwnIP);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'An unexpected error occurred';
    showError(message);
  }
}

function handleFormSubmit(event: Event): void {
  event.preventDefault();
  
  const input = getElementById<HTMLInputElement>('ip-input');
  const ip = input?.value.trim();
  
  handleLookup(ip || undefined);
}

// Get IP from URL query parameter
function getIPFromURL(): string | undefined {
  const params = new URLSearchParams(window.location.search);
  const ip = params.get('ip');
  return ip || undefined;
}

// Initialize
async function init(): Promise<void> {
  // Fetch feature flags first
  featureFlags = await fetchFeatures();
  
  const form = getElementById<HTMLFormElement>('lookup-form');
  if (form) {
    form.addEventListener('submit', handleFormSubmit);
  }
  
  // Check for IP in URL, otherwise auto-lookup current IP
  const ipFromURL = getIPFromURL();
  
  // Pre-fill input if IP is in URL
  if (ipFromURL) {
    const input = getElementById<HTMLInputElement>('ip-input');
    if (input) {
      input.value = ipFromURL;
    }
  }
  
  handleLookup(ipFromURL);
}

// Start when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
