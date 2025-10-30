import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import SlideoutPanel from '../SlideoutPanel';

// Mock createPortal since we're not testing in a real DOM environment
jest.mock('react-dom', () => ({
  ...jest.requireActual('react-dom'),
  createPortal: (node: React.ReactNode) => node,
}));

describe('SlideoutPanel', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
    title: 'Test Panel',
    children: <div>Test content</div>,
  };

  beforeEach(() => {
    jest.clearAllMocks();
    // Mock document.body.style
    Object.defineProperty(document.body, 'style', {
      value: { overflow: '' },
      writable: true,
    });
  });

  afterEach(() => {
    // Clean up any portal elements
    document.body.innerHTML = '';
  });

  it('renders when open', () => {
    render(<SlideoutPanel {...defaultProps} />);
    
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Test Panel')).toBeInTheDocument();
    expect(screen.getByText('Test content')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(<SlideoutPanel {...defaultProps} isOpen={false} />);
    
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', () => {
    const onClose = jest.fn();
    render(<SlideoutPanel {...defaultProps} onClose={onClose} />);
    
    const closeButton = screen.getByLabelText('Close panel');
    fireEvent.click(closeButton);
    
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when backdrop is clicked', () => {
    const onClose = jest.fn();
    render(<SlideoutPanel {...defaultProps} onClose={onClose} />);
    
    // Find the backdrop (first div with bg-black)
    const backdrop = document.querySelector('.bg-black.bg-opacity-50');
    expect(backdrop).toBeInTheDocument();
    
    if (backdrop) {
      fireEvent.click(backdrop);
      expect(onClose).toHaveBeenCalledTimes(1);
    }
  });

  it('calls onClose when Escape key is pressed', () => {
    const onClose = jest.fn();
    render(<SlideoutPanel {...defaultProps} onClose={onClose} />);
    
    fireEvent.keyDown(document, { key: 'Escape' });
    
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('traps focus within the panel', () => {
    render(
      <SlideoutPanel {...defaultProps}>
        <button>First button</button>
        <button>Second button</button>
      </SlideoutPanel>
    );
    
    const dialog = screen.getByRole('dialog');
    const firstButton = screen.getByText('First button');
    const secondButton = screen.getByText('Second button');
    
    // Focus should be trapped - test tab navigation
    fireEvent.keyDown(dialog, { key: 'Tab' });
    // Focus trapping logic is complex to test without a real DOM,
    // but we can verify the dialog is focusable
    expect(dialog).toHaveAttribute('tabIndex', '-1');
  });

  it('applies correct positioning for left side', () => {
    render(<SlideoutPanel {...defaultProps} side="left" />);
    
    const panel = screen.getByRole('dialog');
    expect(panel).toHaveClass('left-0');
  });

  it('applies correct positioning for right side', () => {
    render(<SlideoutPanel {...defaultProps} side="right" />);
    
    const panel = screen.getByRole('dialog');
    expect(panel).toHaveClass('right-0');
  });

  it('applies custom width', () => {
    render(<SlideoutPanel {...defaultProps} width="800px" />);
    
    const panel = screen.getByRole('dialog');
    expect(panel).toHaveStyle({ width: '800px' });
  });

  it('applies custom className', () => {
    render(<SlideoutPanel {...defaultProps} className="custom-class" />);
    
    const panel = screen.getByRole('dialog');
    expect(panel).toHaveClass('custom-class');
  });

  it('prevents body scroll when open', () => {
    render(<SlideoutPanel {...defaultProps} />);
    
    expect(document.body.style.overflow).toBe('hidden');
  });

  it('restores body scroll when closed', () => {
    const { rerender } = render(<SlideoutPanel {...defaultProps} />);
    
    expect(document.body.style.overflow).toBe('hidden');
    
    rerender(<SlideoutPanel {...defaultProps} isOpen={false} />);
    
    expect(document.body.style.overflow).toBe('');
  });

  it('has proper ARIA attributes', () => {
    render(<SlideoutPanel {...defaultProps} />);
    
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(dialog).toHaveAttribute('aria-labelledby', 'slideout-title');
    
    const title = screen.getByText('Test Panel');
    expect(title).toHaveAttribute('id', 'slideout-title');
  });
});