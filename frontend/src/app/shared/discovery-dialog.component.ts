import { Component, Inject, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { FormsModule } from '@angular/forms';
import { DiscoveryService, ModelsLookupItem } from '../generated';
import {
  Subject,
  debounceTime,
  distinctUntilChanged,
  switchMap,
  of,
  catchError,
  finalize,
} from 'rxjs';

@Component({
  selector: 'app-discovery-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatIconModule,
    MatListModule,
    MatProgressBarModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title class="pt-6!">
      <mat-icon class="mr-2 align-middle text-primary">filter_list</mat-icon>
      {{ data.title || '选择资源' }}
    </h2>

    <mat-dialog-content
      class="flex flex-col pb-4!"
      style="min-width: 350px; max-width: 600px; height: 500px;"
    >
      <!-- Search Field -->
      <div class="pt-2 pb-2">
        <mat-form-field appearance="outline" class="w-full" subscriptSizing="dynamic">
          <mat-label>搜索名称或 ID</mat-label>
          <mat-icon matPrefix class="opacity-50 ml-2 mr-1">search</mat-icon>
          <input
            matInput
            [(ngModel)]="search"
            (ngModelChange)="onSearchChange($event)"
            placeholder="输入关键字..."
            autocomplete="off"
          />
        </mat-form-field>
      </div>

      <!-- Height animated loading indicator -->
      <div
        class="overflow-hidden transition-all duration-300 pointer-events-none"
        [style.height]="isLoading() ? '2px' : '0px'"
        [style.opacity]="isLoading() ? 1 : 0"
      >
        <mat-progress-bar mode="indeterminate" class="h-[2px]!"></mat-progress-bar>
      </div>

      <!-- Result List -->
      <div class="flex-1 overflow-y-auto mt-2 -mx-2">
        <mat-selection-list [multiple]="false" class="animate-in fade-in duration-500">
          @if (data.showAllOption) {
            <mat-list-option
              [value]="{ id: '', name: data.allOptionLabel || '全部' }"
              [selected]="data.currentId === ''"
              (click)="onSelect({ id: '', name: data.allOptionLabel || '全部' })"
              class="mb-0.5 h-auto!"
            >
              <div class="flex items-center gap-3 py-1.5">
                <mat-icon class="text-secondary opacity-40 shrink-0">all_inclusive</mat-icon>
                <div class="flex flex-col min-w-0 leading-tight gap-0.5">
                  <span class="text-sm font-bold text-on-surface truncate">{{
                    data.allOptionLabel || '全部'
                  }}</span>
                  <span class="text-[10px] font-mono text-outline opacity-40 truncate"
                    >ALL_RESOURCES</span
                  >
                  <span class="text-[10px] text-outline opacity-30 truncate"
                    >显示所有项，不进行过滤</span
                  >
                </div>
              </div>
            </mat-list-option>
          }

          @for (item of items(); track item.id) {
            <mat-list-option
              [value]="item"
              [selected]="data.currentId === item.id"
              (click)="onSelect(item)"
              class="mb-0.5 h-auto!"
            >
              <div class="flex items-center gap-3 py-1.5">
                <mat-icon class="text-outline opacity-40 shrink-0">{{
                  item.icon || 'category'
                }}</mat-icon>
                <div class="flex flex-col min-w-0 leading-tight gap-0.5">
                  <span class="text-sm font-bold text-on-surface truncate">{{ item.name }}</span>
                  <span class="text-[10px] font-mono text-outline opacity-50 truncate">{{
                    item.id
                  }}</span>
                  @if (item.description) {
                    <span class="text-[10px] text-outline opacity-30 truncate">{{
                      item.description
                    }}</span>
                  }
                </div>
              </div>
            </mat-list-option>
          }

          @if (items().length === 0 && !isLoading() && search) {
            <div class="py-20 flex flex-col items-center justify-center text-outline opacity-30">
              <mat-icon class="text-5xl h-auto w-auto mb-2">search_off</mat-icon>
              <span class="text-sm">未找到相关结果</span>
            </div>
          }
        </mat-selection-list>
      </div>
    </mat-dialog-content>

    <mat-dialog-actions align="end" class="px-6! pb-6!">
      <button mat-button mat-dialog-close>取消</button>
    </mat-dialog-actions>
  `,
  styles: [
    `
      :host ::ng-deep .mat-mdc-list-option {
        --mdc-list-list-item-container-shape: 12px;
      }
    `,
  ],
})
export class DiscoveryDialogComponent implements OnInit {
  private discoveryService = inject(DiscoveryService);
  private dialogRef = inject(MatDialogRef<DiscoveryDialogComponent>);

  search = '';
  items = signal<ModelsLookupItem[]>([]);
  isLoading = signal(false);
  private searchSubject = new Subject<string>();

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: {
      code: string;
      title?: string;
      currentId?: string;
      showAllOption?: boolean;
      allOptionLabel?: string;
    },
  ) {}

  ngOnInit() {
    this.searchSubject
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((s) => {
          this.isLoading.set(true);
          return this.discoveryService.discoveryLookupGet(this.data.code, s, '', 50).pipe(
            catchError(() => of({ items: [] })),
            finalize(() => this.isLoading.set(false)),
          );
        }),
      )
      .subscribe((res) => this.items.set(res.items || []));

    this.searchSubject.next('');
  }

  onSearchChange(val: string) {
    this.searchSubject.next(val);
  }

  onSelect(item: ModelsLookupItem) {
    this.dialogRef.close(item);
  }
}
