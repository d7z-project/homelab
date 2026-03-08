import {
  Component,
  Inject,
  OnInit,
  ViewChild,
  ElementRef,
  AfterViewInit,
  signal,
  computed,
  inject,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { ModelsStepManifest } from '../../generated';

@Component({
  selector: 'app-processor-selector-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatListModule,
    MatIconModule,
    MatButtonModule,
  ],
  template: `
    <div class="flex flex-col h-[600px] max-h-[80vh] overflow-hidden bg-surface">
      <header
        class="px-6 py-4 border-b border-outline-variant/30 flex justify-between items-center shrink-0"
      >
        <h2 class="text-lg font-bold m-0">选择任务处理器</h2>
        <button mat-icon-button icon-button-center mat-dialog-close>
          <mat-icon>close</mat-icon>
        </button>
      </header>

      <div class="p-4 border-b border-outline-variant/10 shrink-0">
        <mat-form-field appearance="outline" class="w-full no-bottom-hint">
          <mat-label>搜索处理器名称或 ID...</mat-label>
          <input matInput [(ngModel)]="searchQuery" (ngModelChange)="updateFilter()" #searchInput />
          <mat-icon matPrefix class="mr-2 opacity-50">search</mat-icon>
        </mat-form-field>
      </div>

      <div class="flex-1 overflow-y-auto p-2" #scrollContainer>
        <mat-selection-list [multiple]="false" (selectionChange)="onSelect($event)">
          @for (m of filteredManifests(); track m.id) {
            <mat-list-option
              [value]="m.id"
              [selected]="m.id === data.selectedId"
              class="rounded-xl mb-1 h-auto py-2"
            >
              <div class="flex items-center gap-4">
                <div
                  class="w-10 h-10 rounded-full bg-primary/5 text-primary flex items-center justify-center shrink-0 border border-primary/10"
                >
                  <mat-icon class="text-[20px]!">extension</mat-icon>
                </div>
                <div class="flex flex-col min-w-0">
                  <div class="flex items-baseline gap-2">
                    <span class="text-sm font-bold text-primary truncate">{{ m.name }}</span>
                    <span class="text-[9px] font-mono text-outline opacity-50"
                      >标识: {{ m.id }}</span
                    >
                  </div>
                  <p
                    class="text-[11px] text-outline opacity-70 line-clamp-2 leading-relaxed m-0 mt-0.5"
                  >
                    {{ m.description }}
                  </p>
                </div>
              </div>
            </mat-list-option>
          } @empty {
            <div class="py-12 text-center text-outline opacity-40 italic">未找到匹配的处理器</div>
          }
        </mat-selection-list>
      </div>
    </div>
  `,
  styles: [
    `
      :host ::ng-deep .no-bottom-hint .mat-mdc-form-field-subscript-wrapper {
        display: none;
      }
      .mat-mdc-list-option {
        --mdc-list-list-item-container-shape: 12px;
      }
    `,
  ],
})
export class ProcessorSelectorDialogComponent implements OnInit, AfterViewInit {
  @ViewChild('searchInput') searchInput!: ElementRef;
  @ViewChild('scrollContainer') scrollContainer!: ElementRef;

  searchQuery = '';
  filteredManifests = signal<ModelsStepManifest[]>([]);

  constructor(
    public dialogRef: MatDialogRef<ProcessorSelectorDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { manifests: ModelsStepManifest[]; selectedId?: string },
  ) {}

  ngOnInit() {
    this.updateFilter();
  }

  ngAfterViewInit() {
    setTimeout(() => {
      this.searchInput.nativeElement.focus();
      this.scrollToSelected();
    }, 100);
  }

  updateFilter() {
    const q = this.searchQuery.toLowerCase();
    this.filteredManifests.set(
      this.data.manifests.filter(
        (m) => m.id?.toLowerCase().includes(q) || m.name?.toLowerCase().includes(q),
      ),
    );
  }

  scrollToSelected() {
    if (!this.data.selectedId) return;
    setTimeout(() => {
      const selectedEl = this.scrollContainer.nativeElement.querySelector(
        '.mat-mdc-list-option.mdc-list-item--selected',
      );
      if (selectedEl) {
        selectedEl.scrollIntoView({ block: 'center', behavior: 'smooth' });
      }
    }, 0);
  }

  onSelect(event: any) {
    const selectedId = event.options[0].value;
    this.dialogRef.close(selectedId);
  }
}
