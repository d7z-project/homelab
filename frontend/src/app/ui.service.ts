import { Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class UiService {
  sidenavOpened = signal(false);

  toggleSidenav() {
    this.sidenavOpened.update((v) => !v);
  }

  setSidenav(opened: boolean) {
    this.sidenavOpened.set(opened);
  }
}
