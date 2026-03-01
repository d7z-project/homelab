import { Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class UiService {
  sidenavOpened = signal(false);
  
  // Toolbar controls
  toolbarVisible = signal(true);
  toolbarShadow = signal(true);
  toolbarDivider = signal(false);
  toolbarSticky = signal(true); // Default to sticky

  toggleSidenav() {
    this.sidenavOpened.update((v) => !v);
  }

  setSidenav(opened: boolean) {
    this.sidenavOpened.set(opened);
  }

  // Helper to reset to defaults
  resetToolbar() {
    this.toolbarVisible.set(true);
    this.toolbarShadow.set(true);
    this.toolbarDivider.set(false);
    this.toolbarSticky.set(true);
  }
}
