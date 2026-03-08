import { Component, Inject, inject, signal, OnInit, ViewChild, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormsModule } from '@angular/forms';
import { MatDialogRef, MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTableModule } from '@angular/material/table';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NetworkSiteService, ModelsSiteGroup, ModelsSitePoolEntry } from '../../generated';
import { Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';

@Component({
  selector: 'app-manage-site-entries-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    FormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatTableModule,
    MatIconModule,
    MatToolbarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
  ],
  template: `
    <div class="flex flex-col h-full bg-surface">
      <mat-toolbar color="primary" class="shrink-0 flex justify-between px-4">
        <div class="flex items-center gap-4">
          <button mat-icon-button mat-dialog-close>
            <mat-icon>arrow_back</mat-icon>
          </button>
          <span class="text-lg font-medium">管理域名池 - {{ data.pool.name }}</span>
        </div>
      </mat-toolbar>

      <div
        class="flex-1 flex flex-col overflow-hidden relative p-4 sm:p-6 max-w-7xl mx-auto w-full"
      >
        <!-- Entry Form -->
        <div
          class="p-6 bg-surface-container-low border border-outline-variant/30 rounded-2xl shrink-0 mb-6 shadow-sm"
        >
          <h3 class="text-sm font-bold uppercase tracking-widest text-primary mb-4">
            {{ isEditMode() ? '修改标签' : '添加新规则' }}
          </h3>
          <form
            [formGroup]="form"
            class="flex flex-col sm:flex-row items-start sm:items-center gap-4"
          >
            <mat-form-field appearance="outline" class="w-full sm:w-40">
              <mat-label>类型</mat-label>
              <mat-select formControlName="type" [disabled]="isEditMode()">
                <mat-option [value]="0">Keyword</mat-option>
                <mat-option [value]="1">Regex</mat-option>
                <mat-option [value]="2">Domain</mat-option>
                <mat-option [value]="3">Full</mat-option>
              </mat-select>
            </mat-form-field>

            <mat-form-field appearance="outline" class="flex-1 w-full">
              <mat-label>规则值 (Value)</mat-label>
              <input
                matInput
                formControlName="value"
                placeholder="例如 google.com"
                [readonly]="isEditMode()"
              />
            </mat-form-field>

            <mat-form-field appearance="outline" class="flex-1 w-full" subscriptSizing="dynamic">
              <mat-label>标签 (Tags)</mat-label>
              <input matInput formControlName="tags" placeholder="逗号分隔" />
              <mat-hint>下划线开头的标签为系统保留，不可在此添加或修改</mat-hint>
              <mat-error *ngIf="form.get('tags')?.hasError('internalTag')">
                标签不能以下划线 '_' 开头
              </mat-error>
            </mat-form-field>

            <div class="flex gap-2 w-full sm:w-auto mt-2 sm:mt-0">
              @if (isEditMode()) {
                <button mat-button type="button" (click)="cancelEdit()">取消</button>
              }
              <button
                mat-flat-button
                color="primary"
                class="h-[56px] px-8"
                [disabled]="form.invalid || submitting()"
                (click)="submit()"
              >
                {{ isEditMode() ? '保存' : '添加' }}
              </button>
            </div>
          </form>
        </div>

        <!-- Search Bar -->
        <div class="mb-4">
          <mat-form-field appearance="outline" class="w-full sm:w-96">
            <mat-label>搜索规则或标签</mat-label>
            <input
              matInput
              [(ngModel)]="searchQuery"
              (ngModelChange)="onSearchChange($event)"
              placeholder="输入关键字..."
            />
            <mat-icon matPrefix>search</mat-icon>
          </mat-form-field>
        </div>

        <!-- Table -->
        <div
          class="flex-1 overflow-auto border border-outline-variant/30 rounded-xl relative bg-surface"
          (scroll)="onScroll($event)"
        >
          @if (loading() && entries().length === 0) {
            <div
              class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-surface/80"
            >
              <mat-spinner diameter="40"></mat-spinner>
            </div>
          }
          <table mat-table [dataSource]="entries()" class="w-full">
            <ng-container matColumnDef="type">
              <th mat-header-cell *matHeaderCellDef class="font-bold">类型</th>
              <td mat-cell *matCellDef="let element">
                <span
                  class="text-[10px] font-bold uppercase px-1.5 py-0.5 rounded border border-outline-variant"
                >
                  @switch (element.type) {
                    @case (0) {
                      kw
                    }
                    @case (1) {
                      re
                    }
                    @case (2) {
                      dom
                    }
                    @case (3) {
                      full
                    }
                  }
                </span>
              </td>
            </ng-container>

            <ng-container matColumnDef="value">
              <th mat-header-cell *matHeaderCellDef class="font-bold">规则值</th>
              <td mat-cell *matCellDef="let element" class="font-mono text-sm">
                {{ element.value }}
              </td>
            </ng-container>

            <ng-container matColumnDef="tags">
              <th mat-header-cell *matHeaderCellDef class="font-bold">标签</th>
              <td mat-cell *matCellDef="let element">
                <div class="flex flex-wrap gap-1 py-1">
                  @for (t of element.tags; track t) {
                    <span
                      class="px-2 py-0.5 rounded border text-[10px] font-bold uppercase tracking-tight"
                      [class.bg-primary/10]="!t.startsWith('_')"
                      [class.border-primary/20]="!t.startsWith('_')"
                      [class.text-primary]="!t.startsWith('_')"
                      [class.bg-surface-container-high]="t.startsWith('_')"
                      [class.text-outline]="t.startsWith('_')"
                      [class.border-outline-variant]="t.startsWith('_')"
                      [matTooltip]="t.startsWith('_') ? '系统保留标签' : ''"
                      >{{ t | uppercase }}</span
                    >
                  }
                  @if (!element.tags || element.tags.length === 0) {
                    <span class="text-outline/30 text-[10px] italic">未设置标签</span>
                  }
                </div>
              </td>
            </ng-container>

            <ng-container matColumnDef="actions">
              <th mat-header-cell *matHeaderCellDef class="w-[100px] font-bold">操作</th>
              <td mat-cell *matCellDef="let element">
                <button
                  mat-icon-button
                  color="primary"
                  (click)="editEntry(element)"
                  matTooltip="修改标签"
                >
                  <mat-icon class="text-sm!">edit</mat-icon>
                </button>
                <button
                  mat-icon-button
                  color="warn"
                  (click)="deleteEntry(element)"
                  matTooltip="删除"
                >
                  <mat-icon class="text-sm!">delete</mat-icon>
                </button>
              </td>
            </ng-container>

            <tr
              mat-header-row
              *matHeaderRowDef="['type', 'value', 'tags', 'actions']; sticky: true"
              class="bg-surface-container-low"
            ></tr>
            <tr
              mat-row
              *matRowDef="let row; columns: ['type', 'value', 'tags', 'actions']"
              class="hover:bg-on-surface/5 transition-colors"
            ></tr>
          </table>

          @if (loadingMore()) {
            <div class="py-4 flex justify-center"><mat-spinner diameter="30"></mat-spinner></div>
          }
        </div>
      </div>
    </div>
  `,
})
export class ManageSiteEntriesDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private siteService = inject(NetworkSiteService);
  private snackBar = inject(MatSnackBar);

  entries = signal<ModelsSitePoolEntry[]>([]);
  loading = signal(false);
  loadingMore = signal(false);
  submitting = signal(false);
  isEditMode = signal(false);

  nextCursor = 0;
  hasMore = signal(true);
  searchQuery = '';
  searchSubject = new Subject<string>();

  form = this.fb.group({
    type: [2, Validators.required],
    value: ['', Validators.required],
    tags: [
      '',
      [
        (control: any) => {
          const val = control.value || '';
          const tags = val.split(',').map((t: string) => t.trim().toLowerCase());
          if (tags.some((t: string) => t.startsWith('_'))) {
            return { internalTag: true };
          }
          return null;
        },
      ],
    ],
  });

  private originalUserTags: string[] = [];

  constructor(
    public dialogRef: MatDialogRef<ManageSiteEntriesDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { pool: ModelsSiteGroup },
  ) {
    this.searchSubject
      .pipe(debounceTime(400), distinctUntilChanged())
      .subscribe(() => this.loadEntries(true));
  }

  ngOnInit() {
    this.loadEntries(true);
  }

  onSearchChange(val: string) {
    this.searchSubject.next(val);
  }

  loadEntries(reset = false) {
    if (reset) {
      this.nextCursor = 0;
      this.hasMore.set(true);
      this.entries.set([]);
      this.loading.set(true);
    } else {
      if (!this.hasMore() || this.loadingMore()) return;
      this.loadingMore.set(true);
    }

    this.siteService
      .networkSitePoolsIdPreviewGet(this.data.pool.id!, this.nextCursor, 50, this.searchQuery)
      .subscribe({
        next: (res) => {
          const newEntries = res.entries || [];
          // 排序标签：带下划线的排在第一位
          newEntries.forEach((entry) => {
            if (entry.tags) {
              entry.tags.sort((a, b) => {
                const aInt = a.startsWith('_');
                const bInt = b.startsWith('_');
                if (aInt && !bInt) return -1;
                if (!aInt && bInt) return 1;
                return a.localeCompare(b);
              });
            }
          });
          if (reset) this.entries.set(newEntries);
          else this.entries.update((v) => [...v, ...newEntries]);
          this.nextCursor = res.nextCursor || 0;
          this.hasMore.set(this.nextCursor > 0 && newEntries.length > 0);
          this.loading.set(false);
          this.loadingMore.set(false);
        },
        error: () => {
          this.loading.set(false);
          this.loadingMore.set(false);
        },
      });
  }

  onScroll(event: Event) {
    const target = event.target as HTMLElement;
    if (target.scrollHeight - target.scrollTop - target.clientHeight < 100) this.loadEntries();
  }

  editEntry(entry: ModelsSitePoolEntry) {
    this.isEditMode.set(true);
    this.originalUserTags = (entry.tags || []).filter((t) => !t.startsWith('_'));
    this.form.patchValue({
      type: entry.type as any,
      value: entry.value,
      tags: this.originalUserTags.join(', '),
    });
  }

  cancelEdit() {
    this.isEditMode.set(false);
    this.originalUserTags = [];
    this.form.reset({ type: 2 });
  }

  submit() {
    if (this.form.invalid) return;
    this.submitting.set(true);
    const val = this.form.getRawValue();
    const newTags = val.tags
      ? val.tags
          .split(',')
          .map((t) => t.trim().toLowerCase())
          .filter((t) => t)
      : [];

    this.siteService
      .networkSitePoolsIdEntriesPost(this.data.pool.id!, {
        type: val.type as any,
        value: val.value!,
        oldTags: this.isEditMode() ? this.originalUserTags : undefined,
        newTags: newTags,
      })
      .subscribe({
        next: () => {
          this.snackBar.open(this.isEditMode() ? '修改成功' : '添加成功', '关闭', {
            duration: 2000,
          });
          this.cancelEdit();
          this.submitting.set(false);
          this.loadEntries(true);
        },
        error: (err) => {
          this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', {
            duration: 3000,
          });
          this.submitting.set(false);
        },
      });
  }

  deleteEntry(entry: ModelsSitePoolEntry) {
    if (!confirm(`确定要删除 ${entry.value} 吗？`)) return;
    this.submitting.set(true);
    this.siteService
      .networkSitePoolsIdEntriesDelete(this.data.pool.id!, entry.type!, entry.value!)
      .subscribe({
        next: () => {
          this.snackBar.open('删除成功', '关闭', { duration: 2000 });
          this.submitting.set(false);
          this.loadEntries(true);
        },
        error: (err) => {
          this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
            duration: 3000,
          });
          this.submitting.set(false);
        },
      });
  }
}
