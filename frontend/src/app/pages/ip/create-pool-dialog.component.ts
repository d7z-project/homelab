import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import { NetworkIpService } from '../../generated';

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
    <h2 mat-dialog-title>新建 IP 地址池</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>池名称</mat-label>
          <input matInput formControlName="name" required />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>描述</mat-label>
          <textarea matInput formControlName="description" rows="3"></textarea>
        </mat-form-field>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="primary" [disabled]="form.invalid || loading" (click)="submit()">
        创建
      </button>
    </mat-dialog-actions>
  `,
})
export class CreatePoolDialogComponent {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private dialogRef = inject(MatDialogRef<CreatePoolDialogComponent>);
  private snackBar = inject(MatSnackBar);

  loading = false;

  form = this.fb.group({
    name: ['', Validators.required],
    description: [''],
  });

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    this.ipService
      .networkIpPoolsPost({
        name: val.name!,
        description: val.description || undefined,
        entryCount: 0,
        checksum: '',
      })
      .subscribe({
        next: () => {
          this.snackBar.open('创建成功', '关闭', { duration: 3000 });
          this.dialogRef.close(true);
        },
        error: (err) => {
          this.loading = false;
          this.snackBar.open(`创建失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
        },
      });
  }
}
