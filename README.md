# Seam Carver

**Seam Carver** is a high-performance content-aware image resizing library implemented in Go. It allows for reducing image width while preserving important visual features by identifying and removing "seams" of low-energy pixels.

## What is Seam Carving?

Seam carving is an algorithm for content-aware image resizing. Unlike standard scaling (which stretches pixels) or cropping (which removes edges), seam carving finds a path of least importance (a "seam") from the top of the image to the bottom and removes it.

This implementation uses a **Gradient Energy Map** to determine pixel importance and **Dynamic Programming** to find the shortest path (lowest energy) across the image.

### Key Features

* **Content-Aware Resizing:** Intelligently reduces image width without distorting subjects.
* **Parallel Energy Calculation:** Utilizes Go routines to calculate pixel gradients concurrently across CPU cores.
* **Batch Processing:** Supports removing multiple seams in a single pass to significantly improve performance on high-resolution images.
* **Dual Modes:** Includes both a simple sequential implementation and a high-performance concurrent implementation.

---

## Getting Started

### Prerequisites

* Go 1.21+ installed on your machine.

### Installation

Clone the repository and navigate to the project root:

```bash
git clone https://github.com/ganimtron-10/seam-carving.git
cd seam-carving

```

### Running the Application

Ensure you have an `images/` directory with a sample `.jpg` or `.png` file, then run:

```bash
go run main.go

```

### Example Usage

The implementation provides two main entry points within `main.go`:

1. **Standard Mode (`mainWithoutConcurrency`):**
* Removes seams one by one.
* Ideal for understanding the core algorithm logic.
* Outputs: `out-img1.jpg`.


2. **Performance Mode (`mainWithConcurrency`):**
* Uses `CalculateEnergyParallel` to distribute workload.
* Uses `RemoveBatchSeams` to delete multiple paths simultaneously, reducing the overhead of re-scanning the image.
* Outputs: `out-img2.jpg`.
