import { Component, OnInit, inject, signal, ViewChild, computed, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatTabsModule } from '@angular/material/tabs';
import { RbacService, AuthServiceAccount, AuthRole, AuthRoleBinding, AuthService } from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatPaginator, MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
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
    MatPaginatorModule,
    MatFormFieldModule,
    MatInputModule,
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
  private uiService = inject(UiService);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: false },
  );

  serviceAccounts = signal<AuthServiceAccount[]>([]);
  roles = signal<AuthRole[]>([]);
  roleBindings = signal<AuthRoleBinding[]>([]);

  saTotal = signal(0);
  roleTotal = signal(0);
  rbTotal = signal(0);

  saPage = signal(0);
  rolePage = signal(0);
  rbPage = signal(0);

  pageSize = signal(20);

  saSearch = signal('');
  roleSearch = signal('');
  rbSearch = signal('');

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);

  hasMoreSa = computed(() => this.serviceAccounts().length < this.saTotal());
  hasMoreRoles = computed(() => this.roles().length < this.roleTotal());
  hasMoreRb = computed(() => this.roleBindings().length < this.rbTotal());

  saColumns: string[] = ['name', 'comments', 'token', 'actions'];
  roleColumns: string[] = ['name', 'rules', 'actions'];
  rbColumns: string[] = ['name', 'sa', 'role', 'actions'];

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

  ngOnInit(): void {
    const params = this.route.snapshot.queryParams;

    if (params['tab'] === 'sa') this.selectedTabIndex.set(0);
    else if (params['tab'] === 'role') this.selectedTabIndex.set(1);
    else if (params['tab'] === 'binding') this.selectedTabIndex.set(2);

    if (params['pageSize']) this.pageSize.set(Number(params['pageSize']));
    if (params['saPage']) this.saPage.set(Number(params['saPage']));
    if (params['saSearch']) this.saSearch.set(params['saSearch']);
    if (params['rolePage']) this.rolePage.set(Number(params['rolePage']));
    if (params['roleSearch']) this.roleSearch.set(params['roleSearch']);
    if (params['rbPage']) this.rbPage.set(Number(params['rbPage']));
    if (params['rbSearch']) this.rbSearch.set(params['rbSearch']);

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
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && !this.loadingMore() && !this.loading()) {
        const tab = this.selectedTabIndex();
        if (tab === 0 && this.hasMoreSa()) {
          this.saPage.update((p) => p + 1);
          this.loadMore('sa');
        } else if (tab === 1 && this.hasMoreRoles()) {
          this.rolePage.update((p) => p + 1);
          this.loadMore('role');
        } else if (tab === 2 && this.hasMoreRb()) {
          this.rbPage.update((p) => p + 1);
          this.loadMore('rb');
        }
      }
    };

    scrollElement.addEventListener('scroll', this.scrollListener);
  }

  private updateQueryParams() {
    const tabs = ['sa', 'role', 'binding'];
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        tab: tabs[this.selectedTabIndex()],
        pageSize: this.pageSize() || null,
        saPage: this.saPage() || null,
        saSearch: this.saSearch() || null,
        rolePage: this.rolePage() || null,
        roleSearch: this.roleSearch() || null,
        rbPage: this.rbPage() || null,
        rbSearch: this.rbSearch() || null,
      },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });
  }

  onTabChange(index: number) {
    this.selectedTabIndex.set(index);
    this.updateQueryParams();
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      this.saPage.set(0);
      this.rolePage.set(0);
      this.rbPage.set(0);
      await Promise.all([this.loadServiceAccounts(true), this.loadRoles(true), this.loadRoleBindings(true)]);
    } catch (err) {
      this.snackBar
        .open('加载数据失败', '重试', { duration: 3000 })
        .onAction()
        .subscribe(() => this.refreshAll());
    } finally {
      setTimeout(() => this.loading.set(false));
    }
  }

  async loadServiceAccounts(reset = false) {
    const data = await firstValueFrom(
      this.rbacService.rbacServiceaccountsGet(this.saPage() + 1, this.pageSize(), this.saSearch()),
    );
    if (reset) {
      this.serviceAccounts.set(data.items || []);
    } else {
      const current = this.serviceAccounts();
      const newItems = (data.items || []).filter(newItem => !current.some(existing => existing.name === newItem.name));
      this.serviceAccounts.update((prev) => [...prev, ...newItems]);
    }
    this.saTotal.set(data.total || 0);
  }

  async loadRoles(reset = false) {
    const data = await firstValueFrom(
      this.rbacService.rbacRolesGet(this.rolePage() + 1, this.pageSize(), this.roleSearch()),
    );
    if (reset) {
      this.roles.set(data.items || []);
    } else {
      const current = this.roles();
      const newItems = (data.items || []).filter(newItem => !current.some(existing => existing.name === newItem.name));
      this.roles.update((prev) => [...prev, ...newItems]);
    }
    this.roleTotal.set(data.total || 0);
  }

  async loadRoleBindings(reset = false) {
    const data = await firstValueFrom(
      this.rbacService.rbacRolebindingsGet(this.rbPage() + 1, this.pageSize(), this.rbSearch()),
    );
    if (reset) {
      this.roleBindings.set(data.items || []);
    } else {
      const current = this.roleBindings();
      const newItems = (data.items || []).filter(newItem => !current.some(existing => existing.name === newItem.name));
      this.roleBindings.update((prev) => [...prev, ...newItems]);
    }
    this.rbTotal.set(data.total || 0);
  }

  onSaSearch(term: string) {
    this.saSearch.set(term);
    this.saPage.set(0);
    this.updateQueryParams();
    this.loadServiceAccounts(true);
  }

  onRoleSearch(term: string) {
    this.roleSearch.set(term);
    this.rolePage.set(0);
    this.updateQueryParams();
    this.loadRoles(true);
  }

  onRbSearch(term: string) {
    this.rbSearch.set(term);
    this.rbPage.set(0);
    this.updateQueryParams();
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
            localStorage.clear();
            this.router.navigate(['/login']);
          },
        });
      }
    });
  }

  async showSaRoles(saName: string) {
    const relevantRbs = this.roleBindings().filter(
      (rb) => rb.serviceAccountName === saName && rb.enabled,
    );
    const roleNames = Array.from(new Set(relevantRbs.flatMap((rb) => rb.roleNames || [])));
    const roles = roleNames
      .map((name) => this.roles().find((r) => r.name === name))
      .filter((r) => !!r) as AuthRole[];

    this.dialog.open(ShowSaRolesDialogComponent, {
      width: '500px',
      data: {
        saName: saName,
        roles: roles,
      },
    });
  }

  editSA(sa: AuthServiceAccount) {
    const dialogRef = this.dialog.open(CreateSaDialogComponent, {
      width: '400px',
      data: { sa: sa, existingNames: this.serviceAccounts().map((x) => x.name) },
    });
    dialogRef.afterClosed().subscribe(async (updatedSa) => {
      if (updatedSa) {
        this.loading.set(true);
        try {
          await firstValueFrom(
            this.rbacService.rbacServiceaccountsNamePut(updatedSa.name, updatedSa),
          );
          this.snackBar.open('ServiceAccount 已更新', '关闭', { duration: 2000 });
          this.saPage.set(0);
          await this.loadServiceAccounts(true);
        } catch (err) {
          this.snackBar.open('更新失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  editRole(role: AuthRole) {
    const dialogRef = this.dialog.open(CreateRoleDialogComponent, {
      width: '500px',
      data: { role: role, existingNames: this.roles().map((x) => x.name) },
    });
    dialogRef.afterClosed().subscribe(async (updatedRole) => {
      if (updatedRole) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolesNamePut(updatedRole.name, updatedRole));
          this.snackBar.open('Role 已更新', '关闭', { duration: 2000 });
          this.rolePage.set(0);
          await this.loadRoles(true);
        } catch (err) {
          this.snackBar.open('更新失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  editRB(binding: AuthRoleBinding) {
    const dialogRef = this.dialog.open(CreateBindingDialogComponent, {
      width: '450px',
      data: {
        serviceAccounts: this.serviceAccounts(),
        roles: this.roles(),
        binding: binding,
        existingNames: this.roleBindings().map((x) => x.name),
      },
    });
    dialogRef.afterClosed().subscribe(async (updatedRB) => {
      if (updatedRB) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolebindingsNamePut(updatedRB.name, updatedRB));
          this.snackBar.open('RoleBinding 已更新', '关闭', { duration: 2000 });
          this.rbPage.set(0);
          await this.loadRoleBindings(true);
        } catch (err) {
          this.snackBar.open('更新失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  createServiceAccount() {
    const dialogRef = this.dialog.open(CreateSaDialogComponent, {
      width: '400px',
      data: { sa: null, existingNames: this.serviceAccounts().map((x) => x.name) },
    });
    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && result.name) {
        this.loading.set(true);
        try {
          const sa = await firstValueFrom(this.rbacService.rbacServiceaccountsPost(result));
          this.snackBar.open('ServiceAccount 已创建', '关闭', { duration: 2000 });
          this.saPage.set(0);
          await this.loadServiceAccounts(true);

          // Show the token to user
          this.dialog.open(ShowTokenDialogComponent, {
            width: '450px',
            data: { name: sa.name, token: sa.token },
            disableClose: true,
          });
        } catch (err) {
          this.snackBar.open('创建失败: ' + (err as any).error?.message || '未知错误', '关闭', {
            duration: 3000,
          });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  deleteSA(name: string) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      data: {
        title: '删除 ServiceAccount',
        message: `确定要永久删除 ServiceAccount "${name}" 吗？此操作不可撤销。`,
        confirmText: '确认删除',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacServiceaccountsNameDelete(name));
          this.snackBar.open('已成功删除 ServiceAccount', '关闭', { duration: 2000 });
          this.saPage.set(0);
          await this.loadServiceAccounts(true);
        } catch (err) {
          this.snackBar.open('删除失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  resetToken(name: string) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      data: {
        title: '重置令牌',
        message: `确定要重置 ServiceAccount "${name}" 的令牌吗？旧令牌将立即失效。`,
        confirmText: '确定重置',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          const sa = await firstValueFrom(this.rbacService.rbacServiceaccountsNameResetPost(name));
          this.snackBar.open('令牌已重置', '关闭', { duration: 2000 });

          // Show the new token
          this.dialog.open(ShowTokenDialogComponent, {
            width: '450px',
            data: { name: sa.name, token: sa.token },
            disableClose: true,
          });
          this.saPage.set(0);
          await this.loadServiceAccounts(true);
        } catch (err) {
          this.snackBar.open('重置失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  createRole() {
    const dialogRef = this.dialog.open(CreateRoleDialogComponent, {
      width: '500px',
      data: { role: null, existingNames: this.roles().map((x) => x.name) },
    });
    dialogRef.afterClosed().subscribe(async (role) => {
      if (role) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolesPost(role));
          this.snackBar.open('Role 已创建', '关闭', { duration: 2000 });
          this.rolePage.set(0);
          await this.loadRoles(true);
        } catch (err) {
          this.snackBar.open('创建失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  deleteRole(name: string) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      data: {
        title: '删除角色 (Role)',
        message: `确定要删除角色 "${name}" 吗？删除后关联的权限绑定可能会失效。`,
        confirmText: '确定删除',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolesNameDelete(name));
          this.snackBar.open('已成功删除角色', '关闭', { duration: 2000 });
          this.rolePage.set(0);
          await this.loadRoles(true);
        } catch (err) {
          this.snackBar.open('删除失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  createRB() {
    const dialogRef = this.dialog.open(CreateBindingDialogComponent, {
      width: '450px',
      data: {
        serviceAccounts: this.serviceAccounts(),
        roles: this.roles(),
        existingNames: this.roleBindings().map((x) => x.name),
      },
    });
    dialogRef.afterClosed().subscribe(async (rb) => {
      if (rb) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolebindingsPost(rb));
          this.snackBar.open('RoleBinding 已创建', '关闭', { duration: 2000 });
          this.rbPage.set(0);
          await this.loadRoleBindings(true);
        } catch (err) {
          this.snackBar.open('创建失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  async toggleRb(rb: AuthRoleBinding) {
    if (!rb.name) return;
    this.loading.set(true);
    try {
      const updated = { ...rb, enabled: !rb.enabled };
      await firstValueFrom(this.rbacService.rbacRolebindingsNamePut(rb.name, updated));
      this.snackBar.open(`绑定已${updated.enabled ? '启用' : '禁用'}`, '关闭', { duration: 2000 });
      this.rbPage.set(0);
      await this.loadRoleBindings(true);
    } catch (err) {
      this.snackBar.open('操作失败', '关闭', { duration: 3000 });
    } finally {
      this.loading.set(false);
    }
  }

  deleteRB(name: string) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      data: {
        title: '解除权限绑定',
        message: `确定要删除绑定 "${name}" 吗？这将立即撤销该 ServiceAccount 的相关权限。`,
        confirmText: '确定解除',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolebindingsNameDelete(name));
          this.snackBar.open('已成功解除绑定', '关闭', { duration: 2000 });
          this.rbPage.set(0);
          await this.loadRoleBindings(true);
        } catch (err) {
          this.snackBar.open('删除失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }
}
