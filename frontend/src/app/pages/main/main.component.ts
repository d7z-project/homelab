import { Component, inject, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  RouterOutlet,
  Router,
  NavigationEnd,
  ActivatedRoute,
  ActivationEnd,
} from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map, filter, tap } from 'rxjs/operators';
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
        padding: 32px 24px;
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        background: transparent;
        color: var(--mat-sys-on-surface);
      }
      .sidebar-logo {
        font-size: 32px;
        height: 32px;
        width: 32px;
        margin-bottom: 16px;
        color: var(--mat-sys-primary);
      }
      .app-sidebar {
        width: 300px;
        border: none !important;
        background-color: var(--mat-sys-surface-container-low) !important;
        border-radius: 0 28px 28px 0 !important;
      }

      mat-nav-list {
        padding: 0 12px !important;
      }

      mat-nav-list a.mat-mdc-list-item {
        border-radius: 28px !important;
        margin-bottom: 4px !important;
        height: 56px !important;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        padding: 0 16px !important;
      }

      .nav-item-active {
        background-color: var(--mat-sys-secondary-container) !important;
        color: var(--mat-sys-on-secondary-container) !important;
      }
      .nav-item-active mat-icon {
        color: var(--mat-sys-on-secondary-container) !important;
      }

      .nav-item-active-parent {
        color: var(--mat-sys-primary) !important;
        font-weight: 600 !important;
      }
      .nav-item-active-parent mat-icon {
        color: var(--mat-sys-primary) !important;
      }

      .sub-menu-container {
        margin-left: 12px;
        padding-left: 8px;
        border-left: 1px solid var(--mat-sys-outline-variant);
      }

      .sidebar-footer {
        position: absolute;
        bottom: 24px;
        width: 100%;
        padding: 0 24px;
        font-size: 10px;
        color: var(--mat-sys-outline);
        text-transform: uppercase;
        letter-spacing: 1px;
        opacity: 0.6;
      }

      /* Expansion Panel M3 Overrides */
      ::ng-deep .mat-expansion-panel-body {
        padding: 0 !important;
      }
      ::ng-deep .mat-expansion-panel-header {
        font-family: inherit !important;
      }
      ::ng-deep .mat-expansion-indicator::after {
        font-size: 18px !important;
      }
    `,
  ],
})
export class MainComponent {
  private breakpointObserver = inject(BreakpointObserver);
  public router = inject(Router);
  public route = inject(ActivatedRoute);
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

    // Reactive handling of viewport changes
    this.breakpointObserver
      .observe(Breakpoints.Handset)
      .pipe(
        map((result) => result.matches),
        tap((handset) => {
          requestAnimationFrame(() => {
            this.uiService.setSidenav(!handset);
          });
        }),
      )
      .subscribe();

    // The MAGIC fix: Listen to route data changes synchronously at the parent level
    this.router.events
      .pipe(
        filter((e) => e instanceof ActivationEnd && e.snapshot.firstChild === null),
        map((e) => (e as ActivationEnd).snapshot.data),
        tap((data) => {
          // Wrap in requestAnimationFrame to ensure the state update happens after the current check cycle,
          // which avoids the NG0100: ExpressionChangedAfterItHasBeenCheckedError.
          requestAnimationFrame(() => {
            const config = data['toolbar'];
            if (config) {
              this.uiService.configureToolbar(config);
            } else {
              this.uiService.resetToolbar();
            }
          });
        }),
      )
      .subscribe();
  }

  menuItems = computed(() => {
    const items: any[] = [
      { link: '/welcome', icon: 'dashboard', label: '控制面板' },
      {
        link: '/rbac',
        icon: 'security',
        label: '权限管理',
        children: [
          { link: '/rbac', queryParams: { tab: 'sa' }, icon: 'account_circle', label: '服务账号' },
          { link: '/rbac', queryParams: { tab: 'role' }, icon: 'shield_person', label: '角色管理' },
          { link: '/rbac', queryParams: { tab: 'binding' }, icon: 'link', label: '权限绑定' },
        ],
      },
      {
        link: '/dns',
        icon: 'dns',
        label: 'DNS 管理',
        children: [
          { link: '/dns', queryParams: { tab: 'domain' }, icon: 'language', label: '域名管理' },
          { link: '/dns', queryParams: { tab: 'record' }, icon: 'layers', label: '解析记录' },
        ],
      },
      {
        link: '/orchestration',
        icon: 'schema',
        label: '任务编排',
        children: [
          {
            link: '/orchestration',
            queryParams: { tab: 'workflow' },
            icon: 'view_kanban',
            label: '工作流模板',
          },
          {
            link: '/orchestration',
            queryParams: { tab: 'instance' },
            icon: 'history',
            label: '运行记录',
          },
        ],
      },
      { link: '/audit', icon: 'history', label: '审计日志' },
    ];

    // Add session management if root
    const rbacItem = items.find((i) => i.link === '/rbac');
    if (rbacItem && rbacItem.children) {
      if (this.uiService.userType() === 'root') {
        rbacItem.children.push({
          link: '/rbac',
          queryParams: { tab: 'session' },
          icon: 'admin_panel_settings',
          label: '管理会话',
        });
      }
      rbacItem.children.push({
        link: '/rbac/simulator',
        icon: 'psychology',
        label: '权限模拟器',
      });
    }

    return items;
  });

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  currentUrl = toSignal(
    this.router.events.pipe(
      filter((e): e is NavigationEnd => e instanceof NavigationEnd),
      map((e) => e.urlAfterRedirects),
    ),
    { initialValue: this.router.url },
  );

  currentPath = computed(() => this.currentUrl().split('?')[0]);

  currentTab = computed(() => {
    const url = this.currentUrl();
    const match = url.match(/[?&]tab=([^&]+)/);
    return match ? match[1] : undefined;
  });

  currentPageLabel = computed(() => {
    const url = this.currentPath();
    const item = this.menuItems().find((m) => {
      const linkPath = m.link.split('?')[0];
      return url === linkPath || url.startsWith(linkPath + '/');
    });
    return item ? item.label : '系统';
  });

  onMenuClick(item: any, event: Event) {
    const targetUrl = item.link;
    const targetParams = item.queryParams || {};

    const currentPath = this.currentPath();
    const currentTab = this.currentTab();

    const isSamePath = currentPath === targetUrl;
    const isSameTab = (!targetParams.tab && !currentTab) || targetParams.tab === currentTab;

    if (isSamePath && isSameTab) {
      event.preventDefault();
      event.stopPropagation();
      if (this.isHandset()) {
        this.uiService.setSidenav(false);
      }
      return;
    }

    this.router.navigate([targetUrl], { queryParams: targetParams });
    if (this.isHandset()) {
      this.uiService.setSidenav(false);
    }
  }

  logout() {
    const dialogRef = this.dialog.open(LogoutDialogComponent, {
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
            localStorage.clear();
            this.router.navigate(['/login']);
          },
        });
      }
    });
  }
}
