import {
  Component,
  Input,
  OnInit,
  forwardRef,
  inject,
  signal,
  ViewChild,
  ElementRef,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  FormsModule,
  ReactiveFormsModule,
  FormControl,
} from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { DiscoveryService, ModelsLookupItem } from '../generated';
import {
  debounceTime,
  distinctUntilChanged,
  switchMap,
  of,
  catchError,
  finalize,
  forkJoin,
} from 'rxjs';

@Component({
  selector: 'app-discovery-select',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    MatChipsModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
  ],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => DiscoverySelectComponent),
      multi: true,
    },
  ],
  template: `
    <mat-form-field [appearance]="appearance" [class]="'w-full relative ' + customClass" [subscriptSizing]="subscriptSizing">
      <mat-label>{{ label }}</mat-label>


      @if (multiple) {
        <mat-chip-grid #chipGrid>
          @for (item of selectedItems(); track item.id) {
            <mat-chip-row (removed)="removeItem(item)" class="!bg-secondary-container">
              <div class="flex flex-col leading-tight py-0.5">
                <span class="text-[10px] font-bold">{{ item.name }}</span>
                @if (item.description) {
                  <span class="text-[8px] opacity-60 truncate max-w-[120px]">{{
                    item.description
                  }}</span>
                }
              </div>
              <button matChipRemove><mat-icon>cancel</mat-icon></button>
            </mat-chip-row>
          }
          <input
            [placeholder]="placeholder"
            [matAutocomplete]="auto"
            [matChipInputFor]="chipGrid"
            [formControl]="inputControl"
            #inputElement
          />
        </mat-chip-grid>
      } @else {
        <input
          matInput
          [placeholder]="placeholder"
          [matAutocomplete]="auto"
          [formControl]="inputControl"
          #inputElement
        />
      }

      <!-- Height animated loading indicator -->
      <div
        class="absolute left-0 right-0 bottom-0 overflow-hidden transition-all duration-300 pointer-events-none"
        [style.height]="isLoading() ? '2px' : '0px'"
        [style.opacity]="isLoading() ? 1 : 0"
      >
        <mat-progress-bar mode="indeterminate" class="!h-[2px]"></mat-progress-bar>
      </div>

      <mat-autocomplete
        #auto="matAutocomplete"
        [displayWith]="displayFn"
        (optionSelected)="onSelected($event)"
      >
        @if (showAllOption) {
          <mat-option [value]="{ id: '', name: allOptionLabel }">
            <div class="flex items-center gap-2 py-1">
              <mat-icon class="!text-sm !w-4 !h-4 opacity-50">all_inclusive</mat-icon>
              <span class="font-bold text-sm">{{ allOptionLabel }}</span>
            </div>
          </mat-option>
        }
        @for (item of items(); track item.id) {
          <mat-option [value]="item" class="!h-auto !py-2 !px-4">
            <div class="flex flex-col leading-tight">
              <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  @if (item.icon) {
                    <mat-icon class="!text-sm !w-4 !h-4 !m-0 opacity-70">{{ item.icon }}</mat-icon>
                  }
                  <span class="font-bold text-[13px]">{{ item.name }}</span>
                </div>
                <span
                  class="text-[9px] font-mono opacity-40 bg-neutral-variant/10 px-1.5 py-0.5 rounded border border-outline-variant/10"
                >
                  {{ item.id }}
                </span>
              </div>
              @if (item.description) {
                <span class="text-[10px] text-outline truncate opacity-70 mt-0.5">
                  {{ item.description }}
                </span>
              }
            </div>
          </mat-option>
        }
        @if (items().length === 0 && !isLoading() && lastSearch()) {
          <mat-option disabled class="text-xs opacity-50">未找到相关项</mat-option>
        }
      </mat-autocomplete>

      @if (hint) {
        <mat-hint>{{ hint }}</mat-hint>
      }
    </mat-form-field>
  `,
})
export class DiscoverySelectComponent implements OnInit, ControlValueAccessor {
  private discoveryService = inject(DiscoveryService);

  @Input() code = '';
  @Input() label = '选择';
  @Input() placeholder = '搜索...';
  @Input() hint = '';
  @Input() appearance: 'fill' | 'outline' = 'outline';
  @Input() subscriptSizing: 'fixed' | 'dynamic' = 'fixed';
  @Input() multiple = false;
  @Input() customClass = '';
  @Input() showAllOption = false;
  @Input() allOptionLabel = '全部';

  @ViewChild('inputElement') inputElement!: ElementRef<HTMLInputElement>;

  inputControl = new FormControl('');
  items = signal<ModelsLookupItem[]>([]);
  selectedItems = signal<ModelsLookupItem[]>([]);
  isLoading = signal(false);
  lastSearch = signal('');
  disabled = false;

  private onChange: (value: any) => void = () => {};
  private onTouched: () => void = () => {};

  ngOnInit() {
    this.inputControl.valueChanges
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((search) => {
          if (typeof search !== 'string') return of(null);
          if (!this.code) return of({ items: [], total: 0 });
          this.isLoading.set(true);
          this.lastSearch.set(search);
          return this.discoveryService.discoveryLookupGet(this.code, search, 0, 20).pipe(
            catchError(() => of({ items: [], total: 0 })),
            finalize(() => this.isLoading.set(false)),
          );
        }),
      )
      .subscribe((res) => {
        if (res) this.items.set(res.items || []);
      });

    // Initial load
    if (this.code) {
      this.discoveryService
        .discoveryLookupGet(this.code, '', 0, 20)
        .subscribe((res) => this.items.set(res.items || []));
    }
  }

  displayFn(item: any): string {
    if (typeof item === 'string') return item;
    return item ? item.name || '' : '';
  }

  onSelected(event: MatAutocompleteSelectedEvent) {
    const item = event.option.value as ModelsLookupItem;
    if (this.multiple) {
      const current = this.selectedItems();
      if (!current.find((i) => i.id === item.id)) {
        this.selectedItems.set([...current, item]);
        this.triggerChange();
      }
      this.inputControl.setValue('', { emitEvent: true });
    } else {
      this.selectedItems.set([item]);
      this.inputControl.setValue(item.name || '', { emitEvent: false });
      this.triggerChange();
    }
  }

  removeItem(item: ModelsLookupItem) {
    this.selectedItems.set(this.selectedItems().filter((i) => i.id !== item.id));
    this.triggerChange();
  }

  private triggerChange() {
    const val = this.multiple
      ? this.selectedItems().map((i) => i.id)
      : this.selectedItems()[0]?.id || '';
    this.onChange(val);
  }

  // ControlValueAccessor implementation
  writeValue(value: any): void {
    if (value === '' && this.showAllOption) {
      this.selectedItems.set([{ id: '', name: this.allOptionLabel }]);
      this.inputControl.setValue(this.allOptionLabel, { emitEvent: false });
      return;
    }

    if (!value || (Array.isArray(value) && value.length === 0)) {
      this.selectedItems.set([]);
      this.inputControl.setValue('', { emitEvent: false });
      return;
    }

    const ids = Array.isArray(value) ? value : [value];
    const currentIds = this.selectedItems().map((i) => i.id);

    // Skip if nothing changed to avoid feedback loops
    if (JSON.stringify(ids) === JSON.stringify(currentIds)) {
      return;
    }

    // Try finding in existing items
    const found = this.items().filter((item) => ids.includes(item.id));
    if (found.length === ids.length) {
      this.selectedItems.set(found);
      if (!this.multiple) {
        this.inputControl.setValue(found[0]?.name || '', { emitEvent: false });
      }
      return;
    }

    // Remote lookup
    if (!this.code) return;
    this.isLoading.set(true);
    const lookups = ids.map((id) =>
      this.discoveryService.discoveryLookupGet(this.code, id, 0, 1).pipe(
        catchError(() => of({ items: [] })),
        switchMap((res) => {
          const item = res.items?.find((i) => i.id === id);
          return of(item);
        }),
      ),
    );

    forkJoin(lookups).subscribe((items) => {
      const validItems = items.filter((i): i is ModelsLookupItem => !!i);
      this.selectedItems.set(validItems);
      if (!this.multiple && validItems.length > 0) {
        this.inputControl.setValue(validItems[0].name || '', { emitEvent: false });
      }
      this.isLoading.set(false);
    });
  }

  registerOnChange(fn: any): void {
    this.onChange = fn;
  }
  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
    if (isDisabled) this.inputControl.disable();
    else this.inputControl.enable();
  }
}
