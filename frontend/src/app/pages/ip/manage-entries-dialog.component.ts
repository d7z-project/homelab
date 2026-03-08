import { Component, Inject, inject, signal, OnInit, ViewChild, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormsModule } from '@angular/forms';
import { MatDialogRef, MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatTableModule } from '@angular/material/table';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDividerModule } from '@angular/material/divider';
import { NetworkIpService, ModelsIPGroup, ModelsIPPoolEntry } from '../../generated';
import { Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { Router } from '@angular/router';

@Component({
  selector: 'app-manage-entries-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    FormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatTableModule,
    MatIconModule,
    MatToolbarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatDividerModule,
  ],
  template: `
    <div class="flex flex-col h-full bg-surface-container-lowest overflow-hidden">
      <!-- Header -->
      <div
        class="bg-surface px-4 py-2 border-b border-outline-variant flex items-center justify-between shrink-0 h-16"
      >
        <div class="flex items-center gap-3">
          <button mat-icon-button mat-dialog-close matTooltip="关闭">
            <mat-icon>arrow_back</mat-icon>
          </button>
          <div class="flex flex-col">
            <span class="text-base font-bold text-primary leading-tight">管理池数据</span>
            <span class="text-[11px] text-outline opacity-70">{{ data.pool.name }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          @if (loading()) {
            <mat-spinner diameter="20"></mat-spinner>
          }
          <button mat-button color="primary" (click)="loadEntries(true)">
            <mat-icon class="mr-1">refresh</mat-icon>刷新
          </button>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto">
        <div class="max-w-6xl mx-auto p-4 sm:p-6 lg:p-8 space-y-6">
          <!-- Add/Edit Form Card -->
          <div
            class="p-6 bg-surface border border-outline-variant rounded-3xl shadow-sm animate-in slide-in-from-top-4 duration-300"
          >
            <div class="flex items-center gap-2 mb-6 text-primary">
              <mat-icon class="w-5! h-5! text-[20px]!">{{
                isEditMode() ? 'edit' : 'add_circle_outline'
              }}</mat-icon>
              <h3 class="text-sm font-bold uppercase tracking-widest">
                {{ isEditMode() ? '修改标签' : '添加记录 / 标签' }}
              </h3>
            </div>

            <form
              [formGroup]="form"
              class="grid grid-cols-1 md:grid-cols-[280px_1fr_auto] gap-4 items-start"
            >
              <mat-form-field appearance="outline" subscriptSizing="dynamic">
                <mat-label>IP 或 CIDR</mat-label>
                <input
                  matInput
                  formControlName="cidr"
                  placeholder="如 192.168.1.1/32"
                  [readonly]="isEditMode()"
                />
              </mat-form-field>

              <mat-form-field appearance="outline" subscriptSizing="dynamic">
                <mat-label>标签 (用逗号分隔多个标签)</mat-label>
                <input matInput formControlName="tags" placeholder="例如：cn, web, office" />
                <mat-hint>下划线开头的标签为系统保留，不可在此添加或修改</mat-hint>
                <mat-error *ngIf="form.get('tags')?.hasError('internalTag')">
                  标签不能以下划线 '_' 开头
                </mat-error>
              </mat-form-field>

              <div class="flex gap-2 h-[56px] items-center">
                @if (isEditMode()) {
                  <button
                    mat-button
                    type="button"
                    (click)="cancelEdit()"
                    class="h-full px-6 rounded-2xl!"
                  >
                    取消
                  </button>
                }
                <button
                  mat-flat-button
                  color="primary"
                  class="h-full px-8 rounded-2xl! shadow-sm"
                  [disabled]="form.invalid || submitting()"
                  (click)="submit()"
                >
                  <div class="flex items-center gap-2">
                    @if (submitting()) {
                      <mat-spinner diameter="18" class="animate-spin"></mat-spinner>
                    }
                    <span>{{ isEditMode() ? '保存修改' : '确认添加' }}</span>
                  </div>
                </button>
              </div>
            </form>
          </div>

          <!-- Data List Card -->
          <div
            class="bg-surface border border-outline-variant rounded-3xl overflow-hidden shadow-sm flex flex-col"
          >
            <!-- Table Toolbar -->
            <div
              class="p-4 bg-surface-container-low border-b border-outline-variant flex flex-wrap items-center justify-between gap-4"
            >
              <div class="flex items-center gap-4 flex-1">
                <mat-form-field
                  appearance="outline"
                  class="w-full sm:w-80 search-field-m3"
                  subscriptSizing="dynamic"
                >
                  <mat-label>搜索 IP 或 标签</mat-label>
                  <input
                    matInput
                    [(ngModel)]="searchQuery"
                    (ngModelChange)="onSearchChange($event)"
                    placeholder="输入关键字..."
                  />
                  <mat-icon matPrefix class="mr-2 opacity-60">search</mat-icon>
                </mat-form-field>
              </div>
              <div class="flex items-center gap-2">
                <span
                  class="px-3 py-1 rounded-full bg-secondary-container text-on-secondary-container text-[10px] font-bold uppercase tracking-wider"
                >
                  共 {{ data.pool.entryCount }} 条
                </span>
              </div>
            </div>

            <!-- Table Container -->
            <div
              class="overflow-auto max-h-[50vh] border-t border-outline-variant/10"
              #scrollContainer
              (scroll)="onScroll($event)"
            >
              <table mat-table [dataSource]="entries()" class="w-full">
                <ng-container matColumnDef="cidr">
                  <th
                    mat-header-cell
                    *matHeaderCellDef
                    class="bg-surface-container-low font-bold text-primary text-center"
                  >
                    IP/CIDR 地址
                  </th>
                  <td mat-cell *matCellDef="let element" class="py-4 text-center">
                    <a
                      class="font-mono text-sm tracking-tight text-primary hover:underline cursor-pointer inline-flex items-center gap-1"
                      (click)="goToAnalysis(element.cidr)"
                      matTooltip="点击在统一研判实验室中分析此地址"
                    >
                      <mat-icon class="w-3! h-3! text-[12px]!">biotech</mat-icon>
                      {{ element.cidr }}
                    </a>
                  </td>
                </ng-container>

                <ng-container matColumnDef="tags">
                  <th
                    mat-header-cell
                    *matHeaderCellDef
                    class="bg-surface-container-low font-bold text-primary text-center"
                  >
                    关联标签
                  </th>
                  <td mat-cell *matCellDef="let element" class="py-4">
                    <div class="flex flex-wrap gap-1.5 justify-center">
                      @for (t of element.tags; track t) {
                        <span
                          class="px-2.5 py-0.5 rounded-full border text-[10px] font-bold uppercase tracking-tighter"
                          [class.bg-primary/10]="!t.startsWith('_')"
                          [class.border-primary/20]="!t.startsWith('_')"
                          [class.text-primary]="!t.startsWith('_')"
                          [class.bg-surface-container-high]="t.startsWith('_')"
                          [class.text-outline]="t.startsWith('_')"
                          [class.border-outline-variant]="t.startsWith('_')"
                          [matTooltip]="t.startsWith('_') ? '系统保留标签' : ''"
                        >
                          {{ t | uppercase }}
                        </span>
                      }
                      @if (!element.tags || element.tags.length === 0) {
                        <span class="text-outline/30 text-[10px] italic">未设置标签</span>
                      }
                    </div>
                  </td>
                </ng-container>

                <ng-container matColumnDef="actions">
                  <th
                    mat-header-cell
                    *matHeaderCellDef
                    class="bg-surface-container-low font-bold text-primary text-center"
                  >
                    管理
                  </th>
                  <td mat-cell *matCellDef="let element" class="text-center py-4">
                    <div class="flex justify-center gap-1">
                      <button
                        mat-icon-button
                        (click)="editEntry(element)"
                        matTooltip="修改标签内容"
                      >
                        <mat-icon class="text-[18px]!">edit_note</mat-icon>
                      </button>
                      <button
                        mat-icon-button
                        color="warn"
                        (click)="deleteEntry(element)"
                        matTooltip="永久删除 (含内部标签记录不可删除)"
                        [disabled]="hasInternalTags(element)"
                      >
                        <mat-icon class="text-[18px]!">delete_sweep</mat-icon>
                      </button>
                    </div>
                  </td>
                </ng-container>

                <tr
                  mat-header-row
                  *matHeaderRowDef="['cidr', 'tags', 'actions']; sticky: true"
                  class="bg-surface-container-low"
                ></tr>
                <tr
                  mat-row
                  *matRowDef="let row; columns: ['cidr', 'tags', 'actions']"
                  class="hover:bg-surface-container-low/50 transition-colors"
                ></tr>

                <tr class="mat-mdc-row" *matNoDataRow>
                  <td
                    class="mat-mdc-cell p-16 text-center text-outline opacity-40 italic"
                    colspan="3"
                  >
                    <div class="flex flex-col items-center gap-2">
                      <mat-icon class="text-6xl! mb-2">inventory_2</mat-icon>
                      <span>暂无匹配的 IP 记录数据</span>
                    </div>
                  </td>
                </tr>
              </table>

              @if (loadingMore()) {
                <div class="p-8 flex justify-center border-t border-outline-variant/20">
                  <mat-spinner diameter="24"></mat-spinner>
                </div>
              }
              @if (!hasMore() && entries().length > 0) {
                <div
                  class="p-8 text-center bg-surface-container-lowest/30 border-t border-outline-variant/20"
                >
                  <div
                    class="flex items-center justify-center gap-2 text-[11px] text-outline font-bold uppercase tracking-widest"
                  >
                    <mat-icon class="w-4! h-4! text-[14px]!">done_all</mat-icon>
                    已加载全部 {{ entries().length }} 条数据
                  </div>
                </div>
              }
            </div>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [
    `
      :host {
        display: block;
        height: 100%;
      }
      .search-field-m3 {
        ::ng-deep .mdc-text-field--filled {
          background-color: transparent !important;
        }
        ::ng-deep .mdc-line-ripple {
          display: none;
        }
        ::ng-deep .mat-mdc-text-field-wrapper {
          padding-bottom: 0;
        }
      }
      ::ng-deep .mat-mdc-header-row {
        background-color: var(--mat-sys-surface-container-low) !important;
        z-index: 10;
      }
      ::ng-deep .mat-mdc-header-cell {
        background-color: inherit !important;
      }
    `,
  ],
})
export class ManageEntriesDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private snackBar = inject(MatSnackBar);
  private router = inject(Router);

  entries = signal<ModelsIPPoolEntry[]>([]);
  loading = signal(false);
  loadingMore = signal(false);
  submitting = signal(false);
  isEditMode = signal(false);

  nextCursor = '';
  hasMore = signal(true);

  searchQuery = '';
  searchSubject = new Subject<string>();

  form = this.fb.group({
    cidr: ['', Validators.required],
    tags: [
      '',
      [
        Validators.required,
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
    public dialogRef: MatDialogRef<ManageEntriesDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { pool: ModelsIPGroup },
  ) {
    this.searchSubject.pipe(debounceTime(400), distinctUntilChanged()).subscribe(() => {
      this.loadEntries(true);
    });
  }

  ngOnInit() {
    this.loadEntries(true);
  }

  onSearchChange(val: string) {
    this.searchSubject.next(val);
  }

  loadEntries(reset = false) {
    if (reset) {
      this.nextCursor = '';
      this.hasMore.set(true);
      this.entries.set([]);
      this.loading.set(true);
    } else {
      if (!this.hasMore() || this.loadingMore()) return;
      this.loadingMore.set(true);
    }

    this.ipService
      .networkIpPoolsIdPreviewGet(this.data.pool.id!, this.nextCursor, 50, this.searchQuery)
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
          if (reset) {
            this.entries.set(newEntries);
          } else {
            this.entries.update((v) => [...v, ...newEntries]);
          }

          this.nextCursor = res.nextCursor || '';
          this.hasMore.set(!!this.nextCursor && newEntries.length > 0);

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
    if (target.scrollHeight - target.scrollTop - target.clientHeight < 100) {
      this.loadEntries();
    }
  }

  goToAnalysis(cidr: string) {
    // 提取 IP 部分，如果有掩码则去掉
    const ip = cidr.includes('/') ? cidr.split('/')[0] : cidr;
    this.dialogRef.close();
    this.router.navigate(['/network/analysis'], { queryParams: { q: ip } });
  }

  editEntry(entry: ModelsIPPoolEntry) {
    this.isEditMode.set(true);
    this.originalUserTags = (entry.tags || []).filter((t) => !t.startsWith('_'));
    this.form.patchValue({
      cidr: entry.cidr,
      tags: this.originalUserTags.join(', '),
    });
  }

  editTag(entry: ModelsIPPoolEntry, tag: string) {
    // 这种模式下，编辑单个标签实际上也是编辑该行所有非内部标签
    this.editEntry(entry);
  }

  cancelEdit() {
    this.isEditMode.set(false);
    this.originalUserTags = [];
    this.form.reset();
  }

  submit() {
    if (this.form.invalid) return;
    this.submitting.set(true);

    const val = this.form.value;
    const newTags = val.tags
      ? val.tags
          .split(',')
          .map((t) => t.trim().toLowerCase())
          .filter((t) => t)
      : [];

    this.ipService
      .networkIpPoolsIdEntriesPost(this.data.pool.id!, {
        cidr: val.cidr!,
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

  deleteEntry(entry: ModelsIPPoolEntry) {
    if (!confirm(`确定要彻底删除 ${entry.cidr} 吗？`)) return;

    this.submitting.set(true);
    this.ipService.networkIpPoolsIdEntriesDelete(this.data.pool.id!, entry.cidr!).subscribe({
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

  hasInternalTags(entry: ModelsIPPoolEntry): boolean {
    return (entry.tags || []).some((t) => t.startsWith('_'));
  }
}
