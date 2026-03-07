import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatIconModule } from '@angular/material/icon';
import { DiscoverySelectComponent } from '../../shared/discovery-select.component';
import { NetworkIpService } from '../../generated';

@Component({
  selector: 'app-create-export-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatIconModule,
    DiscoverySelectComponent,
  ],
  template: `
    <h2 mat-dialog-title>新建导出配置</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>配置名称</mat-label>
          <input matInput formControlName="name" required />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>描述</mat-label>
          <textarea matInput formControlName="description" rows="2"></textarea>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>过滤规则 (go-expr)</mat-label>
          <textarea matInput formControlName="rule" required rows="3" class="font-mono text-sm"></textarea>
          <mat-hint>可用变量: tags ([]string), cidr (string), ip (string). 示例: <code>"cn" in tags || cidr == "8.8.8.8/32"</code></mat-hint>
        </mat-form-field>

        <app-discovery-select
          code="network/ip/pools"
          label="依赖的数据池"
          placeholder="搜索地址池..."
          formControlName="groupIds"
          [multiple]="true"
          required
        ></app-discovery-select>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="primary" [disabled]="form.invalid || loading()" (click)="submit()">
        创建
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateExportDialogComponent {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private dialogRef = inject(MatDialogRef<CreateExportDialogComponent>);
  private snackBar = inject(MatSnackBar);

  loading = signal(false);

  form = this.fb.group({
    name: ['', Validators.required],
    description: [''],
    rule: ['"cn" in tags', Validators.required],
    groupIds: [[] as string[], Validators.required],
  });

  submit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    const val = this.form.value;

    this.ipService
      .networkIpExportsPost({
        name: val.name!,
        description: val.description || undefined,
        rule: val.rule!,
        groupIds: val.groupIds || [],
      })
      .subscribe({
        next: () => {
          this.snackBar.open('创建成功', '关闭', { duration: 3000 });
          this.dialogRef.close(true);
        },
        error: (err) => {
          this.loading.set(false);
          this.snackBar.open(`创建失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
        },
      });
  }
}
