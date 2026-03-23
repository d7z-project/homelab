import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import {
  NetworkIpService,
  ModelsIPPool,
  ModelsIPPoolV1Meta,
  ModelsIPPoolV1Status,
} from '../../generated';

@Component({
  selector: 'app-create-pool-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
  ],
  template: `
    <h2 mat-dialog-title>{{ data.pool ? '修改 IP 地址池' : '新建 IP 地址池' }}</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>地址池 ID</mat-label>
          <input
            matInput
            formControlName="id"
            required
            [readonly]="!!data.pool"
            placeholder="例如: office-lan"
          />
          <mat-hint>仅允许小写字母、数字、中划线和下划线，创建后不可更改</mat-hint>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>池显示名称</mat-label>
          <input matInput formControlName="name" required placeholder="例如: 办公网地址池" />
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
        {{ data.pool ? '保存' : '创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreatePoolDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private dialogRef = inject(MatDialogRef<CreatePoolDialogComponent>);
  private snackBar = inject(MatSnackBar);
  public data = inject(MAT_DIALOG_DATA) as { pool?: ModelsIPPool };

  loading = false;

  form = this.fb.group({
    id: ['', [Validators.required, Validators.pattern(/^[a-z0-9_\-]+$/)]],
    name: ['', Validators.required],
    description: [''],
  });

  ngOnInit() {
    if (this.data.pool) {
      this.form.patchValue({
        id: this.data.pool.id,
        name: this.data.pool.meta?.name,
        description: this.data.pool.meta?.description,
      });
    }
  }

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    const poolData: ModelsIPPool = {
      id: val.id!,
      generation: this.data.pool?.generation || 0,
      meta: {
        name: val.name!,
        description: val.description || undefined,
      },
      status: (this.data.pool?.status || {}) as any,
    };

    const obs = this.data.pool?.id
      ? this.ipService.networkIpPoolsIdPut(this.data.pool.id, poolData)
      : this.ipService.networkIpPoolsPost(poolData);

    obs.subscribe({
      next: () => {
        this.snackBar.open(this.data.pool ? '更新成功' : '创建成功', '关闭', { duration: 3000 });
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.loading = false;
        this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        });
      },
    });
  }
}
