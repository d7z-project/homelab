import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatTabsModule } from '@angular/material/tabs';
import { RbacService, AuthServiceAccount, AuthRole, AuthRoleBinding } from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { firstValueFrom } from 'rxjs';
import { CreateSaDialogComponent } from './create-sa-dialog.component';
import { ShowTokenDialogComponent } from './show-token-dialog.component';
import { CreateRoleDialogComponent } from './create-role-dialog.component';
import { CreateBindingDialogComponent } from './create-binding-dialog.component';
import { ConfirmDialogComponent } from './confirm-dialog.component';

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
  ],
  templateUrl: './rbac.component.html',
})
export class RbacComponent implements OnInit {
  private rbacService = inject(RbacService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);

  serviceAccounts = signal<AuthServiceAccount[]>([]);
  roles = signal<AuthRole[]>([]);
  roleBindings = signal<AuthRoleBinding[]>([]);

  loading = signal(false);
  selectedTabIndex = signal(0);

  saColumns: string[] = ['name', 'comments', 'token', 'actions'];
  roleColumns: string[] = ['name', 'rules', 'actions'];
  rbColumns: string[] = ['name', 'sa', 'role', 'actions'];

  ngOnInit(): void {
    this.refreshAll();
    
    // Handle tab selection from query params
    this.route.queryParams.subscribe(params => {
      const tab = params['tab'];
      if (tab === 'sa') this.selectedTabIndex.set(0);
      else if (tab === 'role') this.selectedTabIndex.set(1);
      else if (tab === 'binding') this.selectedTabIndex.set(2);
    });
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      await Promise.all([
        this.loadServiceAccounts(),
        this.loadRoles(),
        this.loadRoleBindings(),
      ]);
    } catch (err) {
      this.snackBar.open('加载数据失败', '重试', { duration: 3000 }).onAction().subscribe(() => this.refreshAll());
    } finally {
      // Use setTimeout to avoid ExpressionChangedAfterItHasBeenCheckedError
      setTimeout(() => this.loading.set(false));
    }
  }

  async loadServiceAccounts() {
    const data = await firstValueFrom(this.rbacService.rbacServiceaccountsGet());
    this.serviceAccounts.set(data);
  }

  async loadRoles() {
    const data = await firstValueFrom(this.rbacService.rbacRolesGet());
    this.roles.set(data);
  }

  async loadRoleBindings() {
    const data = await firstValueFrom(this.rbacService.rbacRolebindingsGet());
    this.roleBindings.set(data);
  }

  editSA(sa: AuthServiceAccount) {
    const dialogRef = this.dialog.open(CreateSaDialogComponent, { 
      width: '400px',
      data: { sa: sa, existingNames: this.serviceAccounts().map(x => x.name) }
    });
    dialogRef.afterClosed().subscribe(async (updatedSa) => {
      if (updatedSa) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacServiceaccountsPost(updatedSa));
          this.snackBar.open('ServiceAccount 已更新', '关闭', { duration: 2000 });
          await this.loadServiceAccounts();
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
      data: { role: role, existingNames: this.roles().map(x => x.name) }
    });
    dialogRef.afterClosed().subscribe(async (updatedRole) => {
      if (updatedRole) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolesPost(updatedRole));
          this.snackBar.open('Role 已更新', '关闭', { duration: 2000 });
          await this.loadRoles();
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
        existingNames: this.roleBindings().map(x => x.name)
      }
    });
    dialogRef.afterClosed().subscribe(async (updatedRB) => {
      if (updatedRB) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolebindingsPost(updatedRB));
          this.snackBar.open('RoleBinding 已更新', '关闭', { duration: 2000 });
          await this.loadRoleBindings();
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
      data: { sa: null, existingNames: this.serviceAccounts().map(x => x.name) }
    });
    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && result.name) {
        this.loading.set(true);
        try {
          const sa = await firstValueFrom(this.rbacService.rbacServiceaccountsPost(result));
          this.snackBar.open('ServiceAccount 已创建', '关闭', { duration: 2000 });
          await this.loadServiceAccounts();
          
          // Show the token to user
          this.dialog.open(ShowTokenDialogComponent, {
            width: '450px',
            data: { name: sa.name, token: sa.token },
            disableClose: true
          });
        } catch (err) {
          this.snackBar.open('创建失败: ' + (err as any).error?.message || '未知错误', '关闭', { duration: 3000 });
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
        confirmText: '确定删除',
        color: 'warn'
      }
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacServiceaccountsNameDelete(name));
          this.snackBar.open('已成功删除 ServiceAccount', '关闭', { duration: 2000 });
          await this.refreshAll();
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
        color: 'primary'
      }
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
            disableClose: true
          });
          
          await this.loadServiceAccounts();
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
      data: { role: null, existingNames: this.roles().map(x => x.name) }
    });
    dialogRef.afterClosed().subscribe(async (role) => {
      if (role) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolesPost(role));
          this.snackBar.open('Role 已创建', '关闭', { duration: 2000 });
          await this.loadRoles();
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
        color: 'warn'
      }
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolesNameDelete(name));
          this.snackBar.open('已成功删除角色', '关闭', { duration: 2000 });
          await this.refreshAll();
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
        existingNames: this.roleBindings().map(x => x.name)
      }
    });
    dialogRef.afterClosed().subscribe(async (rb) => {
      if (rb) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolebindingsPost(rb));
          this.snackBar.open('RoleBinding 已创建', '关闭', { duration: 2000 });
          await this.loadRoleBindings();
        } catch (err) {
          this.snackBar.open('创建失败', '关闭', { duration: 3000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  deleteRB(name: string) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      data: {
        title: '解除权限绑定',
        message: `确定要删除绑定 "${name}" 吗？这将立即撤销该 ServiceAccount 的相关权限。`,
        confirmText: '确定解除',
        color: 'warn'
      }
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.rbacService.rbacRolebindingsNameDelete(name));
          this.snackBar.open('已成功解除绑定', '关闭', { duration: 2000 });
          await this.loadRoleBindings();
        } catch (err) {
          this.snackBar.open('删除失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }
}
