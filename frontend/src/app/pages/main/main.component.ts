import { Component, inject, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, RouterLink, Router, RouterLinkActive } from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { MatDialog } from '@angular/material/dialog';
import { LogoutDialogComponent } from './logout-dialog.component';
import { AuthService } from '../../generated';

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
  styles: [`
    .nav-item-active {
      background-color: rgba(0, 0, 0, 0.04);
      color: #3f51b5;
      font-weight: 500;
    }
    .nav-item-active mat-icon {
      color: #3f51b5;
    }
    .sidebar-header {
      padding: 16px;
      display: flex;
      flex-direction: column;
      align-items: center;
      background: linear-gradient(45deg, #3f51b5, #5c6bc0);
      color: white;
    }
    .sidebar-logo {
      font-size: 48px;
      height: 48px;
      width: 48px;
      margin-bottom: 8px;
    }
    .app-sidebar {
      width: 256px;
      border-right: 1px solid #e0e0e0 !important;
    }
    ::ng-deep .mat-expansion-panel-body {
      padding: 0 !important;
    }
    ::ng-deep .mat-expansion-panel {
      background: transparent !important;
    }
    ::ng-deep .mat-expansion-panel-header {
      height: 48px !important;
      padding: 0 16px !important;
    }
    ::ng-deep .mat-expansion-panel-header:hover {
      background: rgba(0, 0, 0, 0.04) !important;
    }
    ::ng-deep .mat-content {
      align-items: center !important;
    }
    .submenu-item {
      padding-left: 48px !important;
      height: 40px !important;
      margin: 0 !important;
      border-radius: 0 !important;
    }
    mat-nav-list a.mat-mdc-list-item {
      height: 48px !important;
      margin: 0 !important;
      border-radius: 0 !important;
    }
    .nav-item-active {
      background-color: rgba(63, 81, 181, 0.1) !important;
      color: #3f51b5 !important;
      border-right: 3px solid #3f51b5;
    }
    .nav-item-active mat-icon {
      color: #3f51b5 !important;
    }
    ::ng-deep .mat-mdc-nav-list .mat-mdc-list-item {
      padding-left: 16px !important;
    }
  `]
})
export class MainComponent {
  private breakpointObserver = inject(BreakpointObserver);
  public router = inject(Router);
  private dialog = inject(MatDialog);
  private authService = inject(AuthService);

  constructor() {
    this.authService.infoGet().subscribe({
      error: () => {
        localStorage.clear();
        this.router.navigate(['/login']);
      },
    });
  }

  menuItems = [
    { link: '/welcome', icon: 'dashboard', label: '控制面板' },
    { link: '/rbac', icon: 'security', label: 'RBAC 权限管理' },
  ];

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: false }
  );

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
