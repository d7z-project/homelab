import {
  Component,
  Input,
  OnInit,
  inject,
  signal,
  ViewChild,
  ElementRef,
  Optional,
  Self,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  ControlValueAccessor,
  FormsModule,
  ReactiveFormsModule,
  FormControl,
  NgControl,
  Validator,
  AbstractControl,
  ValidationErrors,
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

/**
 * AppDiscoverySelectComponent
 * Provides a selection-only input (single or multiple) using discovery lookup service.
 */
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
  template: `
    <mat-form-field
      [appearance]="appearance"
      class="w-full relative"
      [subscriptSizing]="subscriptSizing"
    >
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
            (blur)="onTouched()"
            #inputElement
          />
        </mat-chip-grid>
      } @else {
        <input
          matInput
          [placeholder]="placeholder"
          [matAutocomplete]="auto"
          [formControl]="inputControl"
          (blur)="onTouched()"
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
          <mat-option [value]="{ id: '', name: allOptionLabel }" class="!h-auto !py-3">
            <div class="flex items-center gap-4">
              <div
                class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center flex-shrink-0"
              >
                <mat-icon
                  class="!m-0 !text-[24px] !w-6 !h-6 !leading-none flex items-center justify-center text-primary"
                  >all_inclusive</mat-icon
                >
              </div>
              <span class="font-bold text-sm">{{ allOptionLabel }}</span>
            </div>
          </mat-option>
        }
        @for (item of items(); track item.id) {
          <mat-option [value]="item" class="!h-auto !py-3">
            <div class="flex items-start gap-4">
              <!-- Optimized Icon Container with absolute centering -->
              <div
                class="w-10 h-10 rounded-xl bg-surface-container flex items-center justify-center flex-shrink-0 mt-0.5"
              >
                <mat-icon
                  class="!m-0 !text-[24px] !w-6 !h-6 !leading-none flex items-center justify-center opacity-70"
                  >{{ item.icon || 'label' }}</mat-icon
                >
              </div>

              <!-- Content Container -->
              <div class="flex flex-col min-w-0 flex-1 leading-tight">
                <span class="font-bold text-[14px] text-on-surface truncate">{{ item.name }}</span>
                <span class="text-[11px] font-mono text-outline truncate opacity-60 mt-0.5">
                  {{ item.id }}
                </span>
                @if (item.description) {
                  <span
                    class="text-[11px] text-outline truncate opacity-80 mt-1 italic leading-tight"
                  >
                    {{ item.description }}
                  </span>
                }
              </div>
            </div>
          </mat-option>
        }
        @if (items().length === 0 && !isLoading() && lastSearch()) {
          <mat-option disabled class="text-xs opacity-50">未找到相关项</mat-option>
        }
      </mat-autocomplete>

      @if (ngControl && ngControl.errors && ngControl.errors['required']) {
        <mat-error>此项为必填项</mat-error>
      }

      @if (hint && (!ngControl || !ngControl.invalid)) {
        <mat-hint>{{ hint }}</mat-hint>
      }
    </mat-form-field>
  `,
})
export class DiscoverySelectComponent implements OnInit, ControlValueAccessor, Validator {
  private discoveryService = inject(DiscoveryService);

  /** Standard lookup code for discovery service */
  @Input() code = '';
  /** Main field label */
  @Input() label = '选择';
  /** Field placeholder */
  @Input() placeholder = '搜索...';
  /** Optional hint text */
  @Input() hint = '';
  /** Material field appearance */
  @Input() appearance: 'fill' | 'outline' = 'outline';
  /** Subscript sizing behavior */
  @Input() subscriptSizing: 'fixed' | 'dynamic' = 'fixed';
  /** Whether multiple items can be selected */
  @Input() multiple = false;
  /** Whether to show an "All" option with empty ID */
  @Input() showAllOption = false;
  /** Label for the "All" option */
  @Input() allOptionLabel = '全部';

  @ViewChild('inputElement') inputElement!: ElementRef<HTMLInputElement>;

  inputControl = new FormControl('');
  items = signal<ModelsLookupItem[]>([]);
  selectedItems = signal<ModelsLookupItem[]>([]);
  isLoading = signal(false);
  lastSearch = signal('');
  disabled = false;

  constructor(@Optional() @Self() public ngControl: NgControl) {
    if (this.ngControl) {
      this.ngControl.valueAccessor = this;
    }
  }

  public onChange: (value: any) => void = () => {};
  public onTouched: () => void = () => {};

  ngOnInit() {
    // Register validator manually
    if (this.ngControl && this.ngControl.control) {
      this.ngControl.control.addValidators(this.validate.bind(this));
    }

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

  // Validator implementation
  validate(control: AbstractControl): ValidationErrors | null {
    // Basic required check handled by outer [required]
    return null;
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
