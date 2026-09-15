"use client";

import * as React from "react";
import { CalendarIcon } from "lucide-react";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";

import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export interface DatePickerProps extends React.ComponentPropsWithoutRef<typeof Input> {
  value?: Date | string | null;
  onChange?: (date: Date | null) => void;
  placeholder?: string;
  disabled?: boolean;
}

export function DatePicker({
  value,
  onChange,
  placeholder = "Selecionar data",
  disabled,
  className,
  ...props
}: DatePickerProps) {
  const [open, setOpen] = React.useState(false);
  const [selectedDate, setSelectedDate] = React.useState<Date | null>(
    value ? new Date(value) : null,
  );

  React.useEffect(() => {
    if (value) {
      setSelectedDate(new Date(value));
    } else {
      setSelectedDate(null);
    }
  }, [value]);

  const handleSelect = (date: Date | undefined) => {
    if (date) {
      setSelectedDate(date);
      onChange?.(date);
    } else {
      setSelectedDate(null);
      onChange?.(null);
    }
    setOpen(false);
  };

  const handleClear = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedDate(null);
    onChange?.(null);
    setOpen(false);
  };

  const formattedDate = selectedDate ? format(selectedDate, "dd/MM/yyyy", { locale: ptBR }) : "";

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Input
          readOnly
          value={formattedDate}
          placeholder={placeholder}
          disabled={disabled}
          className={cn("cursor-pointer", className)}
          onClick={() => setOpen(!open)}
          {...props}
        />
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start" sideOffset={5}>
        <div className="flex items-center justify-between p-2 border-b">
          <span className="font-medium">Selecionar data</span>
          {selectedDate && (
            <button
              type="button"
              onClick={handleClear}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Limpar
            </button>
          )}
        </div>
        <Calendar
          mode="single"
          selected={selectedDate}
          onSelect={handleSelect}
          initialFocus
          locale={ptBR}
          className="p-2"
        />
      </PopoverContent>
    </Popover>
  );
}
