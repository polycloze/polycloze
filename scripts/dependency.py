"""Provides functions for checking if a target has to be rebuilt."""

from concurrent.futures import Future, ProcessPoolExecutor
from graphlib import TopologicalSorter  # pylint: disable=unused-import
from pathlib import Path
import typing as t


def is_outdated(targets: list[Path], sources: list[Path]) -> bool:
    """Build is outdated if sources timestamp > targets timestamp.

    I.e. inputs are younger than outputs.
    Assumes all inputs exist.
    """
    source_time = max(source.stat().st_mtime_ns for source in sources)
    try:
        target_time = min(target.stat().st_mtime_ns for target in targets)
        return source_time > target_time
    except FileNotFoundError:
        return True


Task = t.Callable[[], t.Any]


def execute(sorter: "TopologicalSorter[Task]") -> None:
    """Execute topologically sorted tasks."""
    futures = []

    sorter.prepare()
    with ProcessPoolExecutor() as executor:
        while sorter.is_active():
            for task in sorter.get_ready():
                def callback(
                    task: Task = task
                ) -> t.Callable[[Future[Task]], None]:
                    return lambda _: sorter.done(task)

                future = executor.submit(task)
                future.add_done_callback(callback())
                futures.append(future)
    for future in futures:
        future.result()
