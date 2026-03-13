import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { NetworkSiteService } from '../../generated';

@Component({
  selector: 'app-create-site-pool-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
  ],
  template: `
    <h2 mat-dialog-title>新建域名资产池</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>资产池 ID</mat-label>
          <input matInput formControlName="id" required placeholder="例如: office-lan" />
          <mat-hint>仅允许小写字母、数字、中划线和下划线，创建后不可更改</mat-hint>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>池显示名称</mat-label>
          <input matInput formControlName="name" required placeholder="例如: 办公网域名池" />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>描述</mat-label>
          <textarea matInput formControlName="description" rows="3"></textarea>
        </mat-form-field>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        [disabled]="form.invalid || loading"
        (click)="submit()"
      >
        创建
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSitePoolDialogComponent {
  private fb = inject(FormBuilder);
  private siteService = inject(NetworkSiteService);
  private dialogRef = inject(MatDialogRef<CreateSitePoolDialogComponent>);
  private snackBar = inject(MatSnackBar);

  loading = false;

  form = this.fb.group({
    id: ['', [Validators.required, Validators.pattern(/^[a-z0-9_\-]+$/)]],
    name: ['', Validators.required],
    description: [''],
  });

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    this.siteService
      .networkSitePoolsPost({
        id: val.id!,
        meta: {
          name: val.name!,
          description: val.description || undefined,
        },
      })
      .subscribe({
        next: () => {
          this.snackBar.open('创建成功', '关闭', { duration: 3000 });
          this.dialogRef.close(true);
        },
        error: (err) => {
          this.loading = false;
          this.snackBar.open(`创建失败: ${err.error?.message || err.message}`, '关闭', {
            duration: 3000,
          });
        },
      });
  }
}
