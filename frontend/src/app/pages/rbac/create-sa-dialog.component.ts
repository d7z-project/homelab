import { Component, Inject, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FormsModule } from '@angular/forms';
import { AuthServiceAccount } from '../../generated';

@Component({
  selector: 'app-create-sa-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">
      {{ isEdit ? '修改 ServiceAccount' : '创建 ServiceAccount' }}
    </h2>
    <mat-dialog-content>
      <div class="pt-3 space-y-5">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>ServiceAccount 名称</mat-label>
          <input
            matInput
            [(ngModel)]="sa.name"
            placeholder="例如: backup-agent"
            [disabled]="isEdit"
            autofocus
            (keyup.enter)="confirm()"
          />
          <mat-hint *ngIf="!isEdit">创建后名称不可修改</mat-hint>
          <mat-error *ngIf="!isEdit && isDuplicate()">名称已存在</mat-error>
        </mat-form-field>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>备注 (Comments)</mat-label>
          <textarea
            matInput
            [(ngModel)]="sa.comments"
            placeholder="说明此账号的用途..."
            rows="3"
          ></textarea>
        </mat-form-field>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        (click)="confirm()"
        [disabled]="!sa.name?.trim() || (!isEdit && isDuplicate())"
        class="!ml-2"
      >
        {{ isEdit ? '保存修改' : '确认创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSaDialogComponent {
  private dialogRef = inject(MatDialogRef<CreateSaDialogComponent>);
  isEdit = false;
  sa: AuthServiceAccount = {
    name: '',
    comments: '',
  };
  existingNames: string[] = [];

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: { sa: AuthServiceAccount | null; existingNames?: string[] },
  ) {
    if (data.sa) {
      this.isEdit = true;
      this.sa = { ...data.sa };
    }
    this.existingNames = data.existingNames || [];
  }

  isDuplicate(): boolean {
    return this.existingNames.includes(this.sa.name?.trim() || '');
  }

  confirm() {
    if (this.sa.name?.trim() && (this.isEdit || !this.isDuplicate())) {
      this.dialogRef.close(this.sa);
    }
  }
}
