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
import { NetworkIpService, ModelsIPGroup, ModelsIPPoolEntry } from '../../generated';
import { Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';

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
  ],
  template: `
    <div class="flex flex-col h-full bg-surface">
      <mat-toolbar color="primary" class="shrink-0 flex justify-between px-4">
        <div class="flex items-center gap-4">
          <button mat-icon-button mat-dialog-close>
            <mat-icon>arrow_back</mat-icon>
          </button>
          <span class="text-lg font-medium">管理池数据 - {{ data.pool.name }}</span>
        </div>
      </mat-toolbar>

      <div class="flex-1 flex flex-col overflow-hidden relative p-4 sm:p-6 max-w-7xl mx-auto w-full">
        <!-- Top Form for adding/updating -->
        <div class="p-6 bg-surface-container-low border border-outline-variant/30 rounded-2xl shrink-0 mb-6 shadow-sm">
          <h3 class="text-sm font-bold uppercase tracking-widest text-primary mb-4">{{ isEditMode() ? '修改记录标签' : '添加新记录' }}</h3>
          <form [formGroup]="form" class="flex flex-col sm:flex-row items-start sm:items-center gap-4">
            <mat-form-field appearance="outline" class="w-full sm:w-64">
              <mat-label>IP 或 CIDR</mat-label>
              <input matInput formControlName="cidr" placeholder="如 192.168.1.1/32" [readonly]="isEditMode()" />
            </mat-form-field>

            <mat-form-field appearance="outline" class="flex-1 w-full">
              <mat-label>标签 (Tags)</mat-label>
              <input matInput formControlName="tags" placeholder="逗号分隔，例如：cn, malicious" />
            </mat-form-field>

            <div class="flex gap-2 w-full sm:w-auto mt-2 sm:mt-0">
              @if (isEditMode()) {
                <button mat-button type="button" (click)="cancelEdit()">取消</button>
              }
              <button mat-flat-button color="primary" class="flex-1 sm:flex-none h-[56px]" [disabled]="form.invalid || submitting()" (click)="submit()">
                {{ isEditMode() ? '保存修改' : '添加' }}
              </button>
            </div>
          </form>
        </div>

        <!-- Search Bar -->
        <div class="mb-4">
          <mat-form-field appearance="outline" class="w-full sm:w-96">
            <mat-label>搜索 IP/CIDR 或 标签</mat-label>
            <input matInput [(ngModel)]="searchQuery" (ngModelChange)="onSearchChange($event)" placeholder="输入关键字..." />
            <mat-icon matPrefix>search</mat-icon>
          </mat-form-field>
        </div>

        <!-- List -->
        <div class="flex-1 overflow-auto border border-outline-variant/30 rounded-xl relative bg-surface" #scrollContainer (scroll)="onScroll($event)">
          @if (loading() && entries().length === 0) {
            <div class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-surface/80 gap-4">
              <mat-spinner diameter="40"></mat-spinner>
              <span class="text-outline text-sm">正在加载数据...</span>
            </div>
          }
          <table mat-table [dataSource]="entries()" class="w-full">
            <ng-container matColumnDef="cidr">
              <th mat-header-cell *matHeaderCellDef class="font-bold">IP/CIDR</th>
              <td mat-cell *matCellDef="let element" class="font-mono text-sm">{{ element.cidr }}</td>
            </ng-container>

            <ng-container matColumnDef="tags">
              <th mat-header-cell *matHeaderCellDef class="font-bold">标签</th>
              <td mat-cell *matCellDef="let element">
                <div class="flex flex-wrap gap-1.5 py-2">
                  @for (t of element.tags; track t) {
                    <span class="px-2 py-0.5 rounded-md bg-primary/10 border border-primary/20 text-primary text-xs font-medium">{{ t }}</span>
                  }
                  @if (!element.tags || element.tags.length === 0) {
                    <span class="text-outline/50 text-xs italic">无标签</span>
                  }
                </div>
              </td>
            </ng-container>

            <ng-container matColumnDef="actions">
              <th mat-header-cell *matHeaderCellDef class="w-[120px] font-bold">操作</th>
              <td mat-cell *matCellDef="let element">
                <button mat-icon-button color="primary" (click)="editEntry(element)" matTooltip="修改标签">
                  <mat-icon class="!text-sm">edit</mat-icon>
                </button>
                <button mat-icon-button color="warn" (click)="deleteEntry(element)" matTooltip="删除记录">
                  <mat-icon class="!text-sm">delete</mat-icon>
                </button>
              </td>
            </ng-container>

            <tr mat-header-row *matHeaderRowDef="['cidr', 'tags', 'actions']; sticky: true" class="bg-surface-container-low"></tr>
            <tr mat-row *matRowDef="let row; columns: ['cidr', 'tags', 'actions'];" class="hover:bg-on-surface/5 transition-colors"></tr>
            
            <tr class="mat-row" *matNoDataRow>
              <td class="mat-cell p-12 text-center text-outline" colspan="3">
                <div class="flex flex-col items-center gap-3">
                  <mat-icon class="!text-4xl opacity-20">find_in_page</mat-icon>
                  <span>暂无匹配的数据</span>
                </div>
              </td>
            </tr>
          </table>
          
          @if (loadingMore()) {
            <div class="py-4 flex justify-center">
              <mat-spinner diameter="30"></mat-spinner>
            </div>
          }
          @if (!hasMore() && entries().length > 0) {
            <div class="py-4 text-center text-xs text-outline/50">
              已加载全部数据
            </div>
          }
        </div>
      </div>
    </div>
  `
})
export class ManageEntriesDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private snackBar = inject(MatSnackBar);

  entries = signal<ModelsIPPoolEntry[]>([]);
  loading = signal(false);
  loadingMore = signal(false);
  submitting = signal(false);
  isEditMode = signal(false);
  
  nextCursor = 0;
  hasMore = signal(true);
  
  searchQuery = '';
  searchSubject = new Subject<string>();

  form = this.fb.group({
    cidr: ['', Validators.required],
    tags: ['']
  });

  constructor(
    public dialogRef: MatDialogRef<ManageEntriesDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { pool: ModelsIPGroup }
  ) {
    this.searchSubject.pipe(
      debounceTime(400),
      distinctUntilChanged()
    ).subscribe(() => {
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
      this.nextCursor = 0;
      this.hasMore.set(true);
      this.entries.set([]);
      this.loading.set(true);
    } else {
      if (!this.hasMore() || this.loadingMore()) return;
      this.loadingMore.set(true);
    }

    this.ipService.networkIpPoolsIdPreviewGet(this.data.pool.id!, this.nextCursor, 50, this.searchQuery).subscribe({
      next: (res) => {
        const newEntries = res.entries || [];
        if (reset) {
          this.entries.set(newEntries);
        } else {
          this.entries.update(v => [...v, ...newEntries]);
        }
        
        this.nextCursor = res.nextCursor || 0;
        this.hasMore.set(this.nextCursor > 0 && newEntries.length > 0);
        
        this.loading.set(false);
        this.loadingMore.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.loadingMore.set(false);
      }
    });
  }

  onScroll(event: Event) {
    const target = event.target as HTMLElement;
    if (target.scrollHeight - target.scrollTop - target.clientHeight < 100) {
      this.loadEntries();
    }
  }

  editEntry(entry: ModelsIPPoolEntry) {
    this.isEditMode.set(true);
    this.form.patchValue({
      cidr: entry.cidr,
      tags: (entry.tags || []).join(', ')
    });
  }

  cancelEdit() {
    this.isEditMode.set(false);
    this.form.reset();
  }

  submit() {
    if (this.form.invalid) return;
    this.submitting.set(true);

    const val = this.form.value;
    const tags = val.tags ? val.tags.split(',').map(t => t.trim()).filter(t => t) : [];

    this.ipService.networkIpPoolsIdEntriesPost(this.data.pool.id!, { cidr: val.cidr!, tags }).subscribe({
      next: () => {
        this.snackBar.open(this.isEditMode() ? '修改成功' : '添加成功', '关闭', { duration: 2000 });
        this.cancelEdit();
        this.submitting.set(false);
        this.loadEntries(true);
      },
      error: (err) => {
        this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
        this.submitting.set(false);
      }
    });
  }

  deleteEntry(entry: ModelsIPPoolEntry) {
    if (!confirm(`确定要删除 ${entry.cidr} 吗？`)) return;
    
    this.submitting.set(true);
    this.ipService.networkIpPoolsIdEntriesDelete(this.data.pool.id!, entry.cidr!).subscribe({
      next: () => {
        this.snackBar.open('删除成功', '关闭', { duration: 2000 });
        this.submitting.set(false);
        this.loadEntries(true);
      },
      error: (err) => {
        this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
        this.submitting.set(false);
      }
    });
  }
}