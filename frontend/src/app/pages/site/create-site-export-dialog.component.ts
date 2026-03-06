import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatIconModule } from '@angular/material/icon';
import { NetworkSiteService, ModelsSiteGroup } from '../../generated';

@Component({
  selector: 'app-create-site-export-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatIconModule,
  ],
  template: `
    <h2 mat-dialog-title>新建域名导出配置</h2>
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
          <mat-hint>可用变量: tags ([]string), domain (string), type (uint8)</mat-hint>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>依赖的域名池</mat-label>
          <mat-select formControlName="groupIds" multiple required>
            @for (pool of pools(); track pool.id) {
              <mat-option [value]="pool.id">{{ pool.name }}</mat-option>
            }
          </mat-select>
        </mat-form-field>
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
export class CreateSiteExportDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private siteService = inject(NetworkSiteService);
  private dialogRef = inject(MatDialogRef<CreateSiteExportDialogComponent>);
  private snackBar = inject(MatSnackBar);

  loading = signal(false);
  pools = signal<ModelsSiteGroup[]>([]);

  form = this.fb.group({
    name: ['', Validators.required],
    description: [''],
    rule: ['"cn" in tags', Validators.required],
    groupIds: [[] as string[], Validators.required],
  });

  ngOnInit() {
    this.siteService.networkSitePoolsGet(1, 1000).subscribe({
      next: (res) => this.pools.set(res.items || []),
    });
  }

  submit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    const val = this.form.value;

    this.siteService
      .networkSiteExportsPost({
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
