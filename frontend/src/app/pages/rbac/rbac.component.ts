import { Component, OnInit, inject, signal, computed, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatTabsModule } from '@angular/material/tabs';
import {
  RbacService,
  ModelsServiceAccount,
  ModelsRole,
  ModelsRoleBinding,
  AuthService,
} from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatDividerModule } from '@angular/material/divider';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { firstValueFrom } from 'rxjs';
import { CreateSaDialogComponent } from './create-sa-dialog.component';
import { ShowTokenDialogComponent } from './show-token-dialog.component';
import { CreateRoleDialogComponent } from './create-role-dialog.component';
import { CreateBindingDialogComponent } from './create-binding-dialog.component';
import { ConfirmDialogComponent } from './confirm-dialog.component';
import { ShowSaRolesDialogComponent } from './show-sa-roles-dialog.component';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { LogoutDialogComponent } from '../main/logout-dialog.component';
import { UiService } from '../../ui.service';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-rbac',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatDialogModule,
    MatTabsModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatMenuModule,
    MatFormFieldModule,
    MatInputModule,
    MatDividerModule,
    MatSlideToggleModule,
    PageHeaderComponent,
  ],
  templateUrl: './rbac.component.html',
})
export class RbacComponent implements OnInit, OnDestroy {
  private rbacService = inject(RbacService);
  private authService = inject(AuthService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private breakpointObserver = inject(BreakpointObserver);
  public uiService = inject(UiService);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  serviceAccounts = signal<ModelsServiceAccount[]>([]);
  roles = signal<ModelsRole[]>([]);
  roleBindings = signal<ModelsRoleBinding[]>([]);

  saTotal = signal(0);
  roleTotal = signal(0);
  rbTotal = signal(0);

  saNextCursor = signal('');
  roleNextCursor = signal('');
  rbNextCursor = signal('');

  hasMoreSa = signal(false);
  hasMoreRoles = signal(false);
  hasMoreRb = signal(false);

  pageSize = signal(20);

  saSearch = signal('');
  roleSearch = signal('');
  rbSearch = signal('');

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  // Enabled toggle column added at the START
  displayedSaColumns = computed(() =>
    this.isHandset()
      ? ['enabled', 'name', 'id', 'actions']
      : ['enabled', 'name', 'id', 'comments', 'lastUsedAt', 'actions'],
  );

  async toggleSa(sa: ModelsServiceAccount) {
    if (!sa.id) return;
    this.loading.set(true);
    try {
      const updated = { ...sa, enabled: sa.enabled! };
      await firstValueFrom(this.rbacService.rbacServiceaccountsIdPut(sa.id, updated));
      this.snackBar.open(`账号已${updated.enabled ? '启用' : '禁用'}`, '关闭', { duration: 2000 });
      // Keep silent refresh or just update local state if preferred, but list reload is safer
      await this.loadServiceAccounts(true);
    } catch (err) {
      // Revert local UI state if reload is not used
      this.snackBar.open('操作失败', '关闭', { duration: 3000 });
    } finally {
      this.loading.set(false);
    }
  }
  displayedRoleColumns = computed(() =>
    this.isHandset() ? ['name', 'actions'] : ['name', 'rules', 'actions'],
  );
  displayedRbColumns = computed(() =>
    this.isHandset()
      ? ['enabled', 'name', 'sa', 'role', 'actions']
      : ['enabled', 'name', 'sa', 'role', 'actions'],
  );

  async toggleRb(rb: ModelsRoleBinding) {
    if (!rb.id) return;
    this.loading.set(true);
    try {
      const updated = { ...rb, enabled: rb.enabled! };
      await firstValueFrom(this.rbacService.rbacRolebindingsIdPut(rb.id, updated));
      this.snackBar.open(`绑定已${updated.enabled ? '启用' : '禁用'}`, '关闭', { duration: 2000 });
      await this.loadRoleBindings(true);
    } catch (err) {
      this.snackBar.open('操作失败', '关闭', { duration: 3000 });
    } finally {
      this.loading.set(false);
    }
  }

  // Helper mappings for RoleBinding display
  getSaName(id: string): string {
    return this.serviceAccounts().find((s) => s.id === id)?.name || id;
  }

  getRoleName(id: string): string {
    return this.roles().find((r) => r.id === id)?.name || id;
  }

  hasSearchContent = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return this.saSearch().length > 0;
    if (tab === 1) return this.roleSearch().length > 0;
    if (tab === 2) return this.rbSearch().length > 0;
    return false;
  });

  openSearch() {
    const tab = this.selectedTabIndex();
    let placeholder = '';
    let value = '';
    let onSearch: (val: string) => void;

    if (tab === 0) {
      placeholder = '搜索账号名称或 ID...';
      value = this.saSearch();
      onSearch = (v) => this.onSaSearch(v);
    } else if (tab === 1) {
      placeholder = '搜索角色名称...';
      value = this.roleSearch();
      onSearch = (v) => this.onRoleSearch(v);
    } else {
      placeholder = '搜索绑定名称或 ID...';
      value = this.rbSearch();
      onSearch = (v) => this.onRbSearch(v);
    }

    this.uiService.openSearch({
      placeholder,
      value,
      onSearch,
    });
  }

  currentSearchTerm = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return this.saSearch();
    if (tab === 1) return this.roleSearch();
    if (tab === 2) return this.rbSearch();
    return '';
  });

  fabConfig = computed(() => {
    switch (this.selectedTabIndex()) {
      case 0:
        return {
          icon: 'add',
          label: '创建账号',
          action: () => this.createServiceAccount(),
        };
      case 1:
        return {
          icon: 'add_moderator',
          label: '新增角色',
          action: () => this.createRole(),
        };
      case 2:
        return {
          icon: 'link',
          label: '建立绑定',
          action: () => this.createRB(),
        };
      default:
        return null;
    }
  });

  constructor() {
    this.route.queryParams.subscribe((params) => {
      if (params['tab'] === 'sa') this.selectedTabIndex.set(0);
      else if (params['tab'] === 'role') this.selectedTabIndex.set(1);
      else if (params['tab'] === 'binding') this.selectedTabIndex.set(2);
    });
  }

  ngOnInit(): void {
    const params = this.route.snapshot.queryParams;
    if (params['pageSize']) this.pageSize.set(Number(params['pageSize']));
    this.refreshAll();
    this.setupScrollListener();
  }

  ngOnDestroy(): void {
    if (this.scrollListener) {
      const scrollElement = document.querySelector('mat-sidenav-content');
      scrollElement?.removeEventListener('scroll', this.scrollListener);
    }
  }

  private setupScrollListener() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (!scrollElement) return;

    this.scrollListener = () => {
      this.showScrollTop.set(scrollElement.scrollTop > 300);
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && !this.loadingMore() && !this.loading()) {
        const tab = this.selectedTabIndex();
        if (tab === 0 && this.hasMoreSa()) {
          this.loadMore('sa');
        } else if (tab === 1 && this.hasMoreRoles()) {
          this.loadMore('role');
        } else if (tab === 2 && this.hasMoreRb()) {
          this.loadMore('rb');
        }
      }
    };
    scrollElement.addEventListener('scroll', this.scrollListener);
  }

  scrollToTop() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (scrollElement) {
      scrollElement.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  private updateQueryParams() {
    const tabs = ['sa', 'role', 'binding'];
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        tab: tabs[this.selectedTabIndex()],
        saSearch: this.saSearch() || null,
        roleSearch: this.roleSearch() || null,
        rbSearch: this.rbSearch() || null,
      },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });
  }

  onTabChange(index: number) {
    this.selectedTabIndex.set(index);
    this.updateQueryParams();

    // Refresh data for the selected tab
    if (index === 0) {
      this.loadServiceAccounts(true);
    } else if (index === 1) {
      this.loadRoles(true);
    } else if (index === 2) {
      this.loadRoleBindings(true);
    }
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      // We load them in order to ensure lookups work
      await this.loadServiceAccounts(true);
      await this.loadRoles(true);
      await this.loadRoleBindings(true);
    } catch (err) {
      this.snackBar
        .open('加载数据失败', '重试', { duration: 3000 })
        .onAction()
        .subscribe(() => this.refreshAll());
    } finally {
      this.loading.set(false);
    }
  }

  async loadServiceAccounts(reset = false) {
    if (reset) {
      this.saNextCursor.set('');
    }
    const data = await firstValueFrom(
      this.rbacService.rbacServiceaccountsGet(
        this.saNextCursor(),
        this.pageSize(),
        this.saSearch(),
      ),
    );
    if (reset) this.serviceAccounts.set(data.items || []);
    else {
      const current = this.serviceAccounts();
      const newItems = (data.items || []).filter(
        (newItem) => !current.some((existing) => existing.id === newItem.id),
      );
      this.serviceAccounts.update((prev) => [...prev, ...newItems]);
    }
    this.saTotal.set(data.total || 0);
    this.saNextCursor.set(data.nextCursor || '');
    this.hasMoreSa.set(data.hasMore || false);
  }

  async loadRoles(reset = false) {
    if (reset) {
      this.roleNextCursor.set('');
    }
    const data = await firstValueFrom(
      this.rbacService.rbacRolesGet(this.roleNextCursor(), this.pageSize(), this.roleSearch()),
    );
    if (reset) this.roles.set(data.items || []);
    else {
      const current = this.roles();
      const newItems = (data.items || []).filter(
        (newItem) => !current.some((existing) => existing.id === newItem.id),
      );
      this.roles.update((prev) => [...prev, ...newItems]);
    }
    this.roleTotal.set(data.total || 0);
    this.roleNextCursor.set(data.nextCursor || '');
    this.hasMoreRoles.set(data.hasMore || false);
  }

  async loadRoleBindings(reset = false) {
    if (reset) {
      this.rbNextCursor.set('');
    }
    const data = await firstValueFrom(
      this.rbacService.rbacRolebindingsGet(this.rbNextCursor(), this.pageSize(), this.rbSearch()),
    );
    if (reset) this.roleBindings.set(data.items || []);
    else {
      const current = this.roleBindings();
      const newItems = (data.items || []).filter(
        (newItem) => !current.some((existing) => existing.id === newItem.id),
      );
      this.roleBindings.update((prev) => [...prev, ...newItems]);
    }
    this.rbTotal.set(data.total || 0);
    this.rbNextCursor.set(data.nextCursor || '');
    this.hasMoreRb.set(data.hasMore || false);
  }

  onSaSearch(term: string) {
    this.saSearch.set(term);
    this.loadServiceAccounts(true);
  }

  onRoleSearch(term: string) {
    this.roleSearch.set(term);
    this.loadRoles(true);
  }

  onRbSearch(term: string) {
    this.rbSearch.set(term);
    this.loadRoleBindings(true);
  }

  async loadMore(type: 'sa' | 'role' | 'rb') {
    this.loadingMore.set(true);
    try {
      if (type === 'sa') await this.loadServiceAccounts();
      else if (type === 'role') await this.loadRoles();
      else if (type === 'rb') await this.loadRoleBindings();
    } finally {
      this.loadingMore.set(false);
    }
  }

  toggleDrawer() {
    this.uiService.toggleSidenav();
  }

  logout() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(LogoutDialogComponent, { maxWidth: '90vw' });
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
    });
  }

  async showSaRoles(sa: ModelsServiceAccount) {
    const saID = sa.id || '';
    const relevantRbs = this.roleBindings().filter(
      (rb) => rb.serviceAccountId === saID && rb.enabled,
    );
    const roleIDs = Array.from(new Set(relevantRbs.flatMap((rb) => rb.roleIds || [])));
    const roles = roleIDs
      .map((id) => this.roles().find((r) => r.id === id))
      .filter((r) => !!r) as ModelsRole[];

    requestAnimationFrame(() => {
      this.dialog.open(ShowSaRolesDialogComponent, {
        data: { saID: saID, saName: sa.name, roles: roles },
      });
    });
  }

  editSA(sa: ModelsServiceAccount) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateSaDialogComponent, {
        data: { sa: sa, existingIDs: this.serviceAccounts().map((x) => x.id || '') },
      });
      dialogRef.afterClosed().subscribe(async (updatedSa) => {
        if (updatedSa && updatedSa.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(
              this.rbacService.rbacServiceaccountsIdPut(updatedSa.id, updatedSa),
            );
            this.snackBar.open('账号已更新', '关闭', { duration: 2000 });
            await this.loadServiceAccounts(true);
          } catch (err) {
            this.snackBar.open('更新失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  editRole(role: ModelsRole) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateRoleDialogComponent, {
        data: { role: role, existingIDs: this.roles().map((x) => x.id || '') },
      });
      dialogRef.afterClosed().subscribe(async (updatedRole) => {
        if (updatedRole && updatedRole.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacRolesIdPut(updatedRole.id, updatedRole));
            this.snackBar.open('角色已更新', '关闭', { duration: 2000 });
            await this.loadRoles(true);
          } catch (err) {
            this.snackBar.open('更新失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  editRB(binding: ModelsRoleBinding) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateBindingDialogComponent, {
        data: {
          binding: binding,
        },
      });
      dialogRef.afterClosed().subscribe(async (updatedRB) => {
        if (updatedRB && updatedRB.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacRolebindingsIdPut(updatedRB.id, updatedRB));
            this.snackBar.open('绑定已更新', '关闭', { duration: 2000 });
            await this.loadRoleBindings(true);
          } catch (err) {
            this.snackBar.open('更新失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  createServiceAccount() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateSaDialogComponent, {
        data: { sa: null, existingIDs: this.serviceAccounts().map((x) => x.id || '') },
      });
      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && result.id) {
          this.loading.set(true);
          try {
            const sa = await firstValueFrom(this.rbacService.rbacServiceaccountsPost(result));
            this.snackBar.open('账号已创建', '关闭', { duration: 2000 });
            await this.loadServiceAccounts(true);

            requestAnimationFrame(() => {
              this.dialog.open(ShowTokenDialogComponent, {
                data: { id: sa.id, name: sa.name, token: sa.token },
                disableClose: true,
              });
            });
          } catch (err) {
            this.snackBar.open('创建失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  deleteSA(sa: ModelsServiceAccount) {
    const id = sa.id || '';
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除服务账号',
          message: `确定要永久删除账号 "${sa.name || id}" 吗？此操作不可撤销。`,
          confirmText: '确认删除',
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacServiceaccountsIdDelete(id));
            this.snackBar.open('账号已删除', '关闭', { duration: 2000 });
            await this.loadServiceAccounts(true);
          } catch (err) {
            this.snackBar.open('删除失败', '关闭', { duration: 2000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  resetToken(sa: ModelsServiceAccount) {
    const id = sa.id || '';
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '重置令牌',
          message: `确定要重置账号 "${sa.name || id}" 的令牌吗？旧令牌将立即失效。`,
          confirmText: '确定重置',
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            const res = await firstValueFrom(this.rbacService.rbacServiceaccountsIdResetPost(id));
            this.snackBar.open('令牌已重置', '关闭', { duration: 2000 });
            requestAnimationFrame(() => {
              this.dialog.open(ShowTokenDialogComponent, {
                data: { id: res.id, name: res.name, token: res.token },
                disableClose: true,
              });
            });
            await this.loadServiceAccounts(true);
          } catch (err) {
            this.snackBar.open('重置失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  createRole() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateRoleDialogComponent, {
        data: { role: null, existingIDs: this.roles().map((x) => x.id || '') },
      });
      dialogRef.afterClosed().subscribe(async (role) => {
        if (role) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacRolesPost(role));
            this.snackBar.open('角色已创建', '关闭', { duration: 2000 });
            await this.loadRoles(true);
          } catch (err) {
            this.snackBar.open('创建失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  deleteRole(role: ModelsRole) {
    const id = role.id || '';
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除角色',
          message: `确定要删除角色 "${role.name || id}" 吗？删除后关联的权限绑定可能会失效。`,
          confirmText: '确定删除',
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacRolesIdDelete(id));
            this.snackBar.open('角色已删除', '关闭', { duration: 2000 });
            await this.loadRoles(true);
          } catch (err) {
            this.snackBar.open('删除失败', '关闭', { duration: 2000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  createRB() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateBindingDialogComponent, {
        data: {},
      });
      dialogRef.afterClosed().subscribe(async (rb) => {
        if (rb) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacRolebindingsPost(rb));
            this.snackBar.open('绑定已创建', '关闭', { duration: 2000 });
            await this.loadRoleBindings(true);
          } catch (err) {
            this.snackBar.open('创建失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  deleteRB(rb: ModelsRoleBinding) {
    const id = rb.id || '';
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '解除绑定',
          message: `确定要删除绑定 "${rb.name || id}" 吗？这将立即撤销该账号的相关权限。`,
          confirmText: '确定解除',
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.rbacService.rbacRolebindingsIdDelete(id));
            this.snackBar.open('已成功解除绑定', '关闭', { duration: 2000 });
            await this.loadRoleBindings(true);
          } catch (err) {
            this.snackBar.open('删除失败', '关闭', { duration: 2000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }
}
