import { Component, inject, signal, effect, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, RouterLink, Router, RouterLinkActive, NavigationEnd } from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map, filter } from 'rxjs/operators';
import { MatDialog } from '@angular/material/dialog';
import { LogoutDialogComponent } from './logout-dialog.component';
import { AuthService } from '../../generated';
import { UiService } from '../../ui.service';

@Component({
  selector: 'app-main',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    MatSidenavModule,
    MatListModule,
    MatIconModule,
    MatToolbarModule,
    MatButtonModule,
    MatDividerModule,
    MatExpansionModule,
  ],
  templateUrl: './main.component.html',
  styles: [
    `
      .sidebar-header {
        padding: 24px 16px;
        display: flex;
        flex-direction: column;
        align-items: center;
        background: var(--mat-sys-surface-container-low);
        color: var(--mat-sys-on-surface);
        border-radius: 0 0 24px 24px;
        margin-bottom: 8px;
      }
      .sidebar-logo {
        font-size: 40px;
        height: 40px;
        width: 40px;
        margin-bottom: 12px;
        color: var(--mat-sys-primary);
      }
      .app-sidebar {
        width: 280px;
        border: none !important;
        background-color: var(--mat-sys-surface-container-low) !important;
      }

      header {
        height: 64px;
      }

      mat-nav-list {
        padding: 12px !important;
      }

      mat-nav-list a.mat-mdc-list-item {
        border-radius: 28px !important;
        margin-bottom: 4px !important;
        height: 56px !important;
        transition: all 0.2s ease-in-out;
      }

      .nav-item-active {
        background-color: var(--mat-sys-secondary-container) !important;
        color: var(--mat-sys-on-secondary-container) !important;
      }

      .nav-item-active mat-icon {
        color: var(--mat-sys-on-secondary-container) !important;
      }

      .sidebar-footer {
        position: absolute;
        bottom: 16px;
        width: 100%;
        text-align: center;
        padding: 0 16px;
        font-size: 11px;
        color: var(--mat-sys-outline);
        text-transform: uppercase;
        letter-spacing: 0.5px;
      }
    `,
  ],
})
export class MainComponent {
  private breakpointObserver = inject(BreakpointObserver);
  public router = inject(Router);
  private dialog = inject(MatDialog);
  private authService = inject(AuthService);
  public uiService = inject(UiService);

  constructor() {
    this.authService.infoGet().subscribe({
      error: () => {
        localStorage.clear();
        this.router.navigate(['/login']);
      },
    });

    // Automatically sync initial state for handset
    effect(() => {
      const handset = this.isHandset();
      if (!handset) {
        this.uiService.setSidenav(true);
      } else {
        this.uiService.setSidenav(false);
      }
    }, { allowSignalWrites: true });
  }

  menuItems = [
    { link: '/welcome', icon: 'dashboard', label: '控制面板' },
    { link: '/rbac', icon: 'security', label: 'RBAC 权限管理' },
  ];

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: false },
  );

  currentUrl = toSignal(
    this.router.events.pipe(
      filter((e): e is NavigationEnd => e instanceof NavigationEnd),
      map((e) => e.urlAfterRedirects),
    ),
    { initialValue: this.router.url },
  );

  currentPageLabel = computed(() => {
    const url = this.currentUrl().split('?')[0];
    const item = this.menuItems.find((m) => {
      const linkPath = m.link.split('?')[0];
      return url === linkPath || url.startsWith(linkPath + '/');
    });
    return item ? item.label : '系统';
  });

  logout() {
    const dialogRef = this.dialog.open(LogoutDialogComponent, {
      width: '400px',
      maxWidth: '90vw',
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.authService.logoutPost().subscribe({
          next: () => {
            localStorage.clear();
            this.router.navigate(['/login']);
          },
          error: () => {
            // Even if backend fails, we should clear local storage and redirect
            localStorage.clear();
            this.router.navigate(['/login']);
          },
        });
      }
    });
  }
}
