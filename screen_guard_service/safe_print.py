"""
Safe print utility for Windows console compatibility
Handles Unicode emoji encoding errors gracefully
"""

import sys


def safe_print(*args, **kwargs):
    """
    Print function that handles encoding errors gracefully on Windows.
    Proactively replaces emojis with ASCII alternatives for Windows compatibility.
    """
    # Proactively replace emojis with ASCII alternatives for Windows compatibility
    safe_args = []
    for arg in args:
        if isinstance(arg, str):
            # Replace common emojis with ASCII alternatives
            arg = arg.replace('🚨', '[ALERT]')
            arg = arg.replace('⚠️', '[WARN]')
            arg = arg.replace('🔧', '[INIT]')
            arg = arg.replace('✓', '[OK]')
            arg = arg.replace('✅', '[OK]')
            arg = arg.replace('🔍', '[SCAN]')
            arg = arg.replace('❌', '[ERROR]')
            arg = arg.replace('📊', '[STATS]')
            arg = arg.replace('🛑', '[STOP]')
            arg = arg.replace('🔴', '[CRITICAL]')
            arg = arg.replace('📋', '[INFO]')
            arg = arg.replace('ℹ️', '[INFO]')
            arg = arg.replace('▶️', '[START]')
            arg = arg.replace('⏸️', '[PAUSE]')
            arg = arg.replace('🎥', '[VIDEO]')
            arg = arg.replace('📹', '[CAMERA]')
            arg = arg.replace('📵', '[STOP]')
            arg = arg.replace('🤖', '[BOT]')
            arg = arg.replace('⏳', '[WAIT]')
            arg = arg.replace('🌙', '[NIGHT]')
            arg = arg.replace('🔔', '[NOTIFY]')
        safe_args.append(arg)
    try:
        print(*safe_args, **kwargs)
    except UnicodeEncodeError:
        # Fallback: try to encode and replace any remaining problematic characters
        final_args = []
        for arg in safe_args:
            if isinstance(arg, str):
                # Remove any remaining non-ASCII characters that can't be encoded
                arg = arg.encode('ascii', 'replace').decode('ascii')
            final_args.append(arg)
        print(*final_args, **kwargs)

