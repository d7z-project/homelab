import { Injectable, signal } from '@angular/core';

export interface SearchConfig {
  placeholder: string;
  value: string;
  onSearch: (value: string) => void;
  onClose?: () => void;
}

@Injectable({
  providedIn: 'root',
})
export class UiService {
  // Global loading state for initial app check
  initializing = signal(true);
  userType = signal<string | null>(null);
  sessionId = signal<string | null>(null);

  // Sidenav state
  sidenavOpened = signal(false);

  // Dynamic toolbar controls (inherited from route or manually set)
  toolbarVisible = signal(true);
  toolbarShadow = signal(false);
  toolbarDivider = signal(false);
  toolbarSticky = signal(false);

  // Global Search Overlay state
  searchConfig = signal<SearchConfig | null>(null);

  openSearch(config: SearchConfig) {
    this.searchConfig.set(config);
  }

  closeSearch() {
    const current = this.searchConfig();
    if (current && current.onClose) {
      current.onClose();
    }
    this.searchConfig.set(null);
  }

  toggleSidenav() {
    this.sidenavOpened.update((v) => !v);
  }

  setSidenav(opened: boolean) {
    this.sidenavOpened.set(opened);
  }

  configureToolbar(config: {
    visible?: boolean;
    shadow?: boolean;
    divider?: boolean;
    sticky?: boolean;
  }) {
    if (config.visible !== undefined) this.toolbarVisible.set(config.visible);
    if (config.shadow !== undefined) this.toolbarShadow.set(config.shadow);
    if (config.divider !== undefined) this.toolbarDivider.set(config.divider);
    if (config.sticky !== undefined) this.toolbarSticky.set(config.sticky);
  }
}
